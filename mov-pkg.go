package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return fmt.Sprint(*i)
}

func (i *arrayFlags) Set(value string) error {
	for _, dt := range strings.Split(value, ",") {
		*i = append(*i, dt)
	}
	return nil
}

type App struct {
	Fset         *token.FileSet
	PkgFrom      string
	NamesLocated string
	NamesFlags   arrayFlags
	NamesFrom    map[string]string
	PkgTo        string
	PkgToName    string
	SuffixTo     string
	IsDryRun     bool
}

type FileChanging struct {
	FileName      string
	AstFile       *ast.File
	Pkg           *packages.Package
	Count         int
	SkipAddImport bool
	ImportNames   map[string]string
}

func NewApp() *App {

	app := &App{
		Fset:      token.NewFileSet(),
		NamesFrom: make(map[string]string),
	}

	flag.StringVar(&app.PkgFrom, "f", "", "Package from, eg. 'github.com/firodj/mov-pkg/examples'")
	flag.StringVar(&app.PkgTo, "t", "", "Package to, eg. 'github.com/firodj/mov-pkg/examples/models'")
	flag.StringVar(&app.NamesLocated, "l", "/models.go", "What File the Type is defined")
	flag.StringVar(&app.PkgToName, "a", "", "Package alias name, left it blank to use default")
	flag.Var(&app.NamesFlags, "n", "The Type name to be rename/move, left it blank to use all. This will combined with where the Type defined")
	flag.StringVar(&app.SuffixTo, "s", "", "Target suffix for Type name")
	flag.BoolVar(&app.IsDryRun, "d", false, "Dry run")
	flag.Parse()

	if app.IsDryRun {
		fmt.Println("Running in Dry Run...")
	}
	if app.PkgFrom == "" {
		fmt.Println("Missing -f params")
		return nil
	} else {
		fmt.Printf("Pkg From: %s\n", app.PkgFrom)
	}
	if app.PkgTo == "" {
		fmt.Println("Missing -t params")
		return nil
	}

	return app
}

var FileChanges map[string]*FileChanging = make(map[string]*FileChanging)
var PkgPackages map[string]*packages.Package = make(map[string]*packages.Package)

func (app *App) MyApplier(cr *astutil.Cursor, fc *FileChanging) bool {
	info := fc.Pkg.TypesInfo

	pkgToName := app.PkgToName
	if pkgToNameThisFile, ok := fc.ImportNames[app.PkgTo]; ok {
		pkgToName = pkgToNameThisFile
	}

	switch node := cr.Node().(type) {
	case *ast.Ident:
		// NOTES: shadowing package name:
		// cr.Parent = *ast.ValueSpec, cr.Name = Names, cr.Node.(*Ident).Name = models,
		// cr.Parent = *ast.AssignStmt, cr.Name = Lhs, cr.Node.(*Ident).Name = models,

		//v := reflect.ValueOf(cr.Parent())
		//fmt.Printf("cr.Parent = %s, ", v.Type().String())
		//fmt.Printf("cr.Name = %s, cr.Node.(*Ident).Name = %s,\n", cr.Name(), node.Name)

		doReplace := false
		if uses, ok := info.Uses[node]; ok {
			//v := reflect.ValueOf(uses)
			// spew.Printf("+ident = %+v, uses = %+v\n", node, v.Type().String())

			if typeName, ok := uses.(*types.TypeName); ok {
				var pkgPath string
				if typeName.Pkg() != nil {
					pkgPath = typeName.Pkg().Path()
				}

				// fmt.Printf("--> %s; %s\n", typeName.Name(), pkgPath)
				if pkgPath == app.PkgFrom {
					if _, ok := app.NamesFrom[typeName.Name()]; ok {
						doReplace = true
					}
				}
			}
		}

		if doReplace {
			/**
			cr.Replace(&ast.Ident{
				Name: "services_" + node.Name,
			})
			 **/
			fc.Count++
			cr.Replace(&ast.SelectorExpr{
				X: &ast.Ident{
					Name: pkgToName,
				},
				Sel: &ast.Ident{
					Name: fmt.Sprintf("%s%s", node.Name, app.SuffixTo),
				},
			})
		}

		return false

	case *ast.SelectorExpr:
		if pkgID, ok := node.X.(*ast.Ident); ok {
			if uses, ok := info.Uses[pkgID]; ok {
				//v := reflect.ValueOf(uses)
				//spew.Printf("+selector.ident = %+v, uses = %+v, Sel = %+#v\n", pkgID, v.Type().String(), node.Sel)

				doReplace := false
				var pkgPath string
				if pkgName, ok := uses.(*types.PkgName); ok {
					pkgPath = pkgName.Imported().Path()

					//fmt.Printf("Selector.X = %s, .Sel = %s\n", pkgPath, node.Sel.Name)
					//fmt.Printf("=> %s\n", pkgName.Imported().Name())

					if selUses, ok := info.Uses[node.Sel]; ok {
						if typeName, ok := selUses.(*types.TypeName); ok {
							//var selPkgPath string
							//if typeName.Pkg() != nil {
							//	selPkgPath = typeName.Pkg().Path()
							//}

							if pkgPath == app.PkgFrom {
								if _, ok := app.NamesFrom[typeName.Name()]; ok {
									doReplace = true
								}
							}

							//fmt.Printf("selPkgPath = %s\n", selPkgPath)
						}
					}
				}

				if doReplace {
					fc.Count++
					cr.Replace(&ast.SelectorExpr{
						X: &ast.Ident{
							Name: pkgToName,
						},
						Sel: &ast.Ident{
							Name: fmt.Sprintf("%s%s", node.Sel.Name, app.SuffixTo),
						},
					})
				}
			}
			return false
		}
	}
	return true
}

func (app *App) LoadPackages() {
	patterns := flag.Args()

	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo,
		Fset:  app.Fset,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load: %v\n", err)
		os.Exit(1)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	// Pick  the names of the source files
	// for each package listed on the command line.
	for _, pkg := range pkgs {
		pkgSubIDs := strings.Split(pkg.ID, " ")
		if strings.HasSuffix(pkgSubIDs[0], ".test") {
			continue
		}

		if _, ok := PkgPackages[pkgSubIDs[0]]; !ok {
			PkgPackages[pkgSubIDs[0]] = pkg
		} else {
			if len(pkgSubIDs) > 1 {
				PkgPackages[pkgSubIDs[0]] = pkg
			}
		}
	}

	// Find proper package name or alise for To
	if app.PkgToName == "" {
		if pkgTo, ok := PkgPackages[app.PkgTo]; ok {
			app.PkgToName = pkgTo.Name
		} else {
			i := strings.LastIndex(app.PkgTo, "/")
			app.PkgToName = app.PkgTo[i+1:]
		}
	}

	fmt.Printf("Pkg To: %s %s\n", app.PkgToName, app.PkgTo)
}

func (app *App) ListDefines() {
	for _, pkg := range PkgPackages {

		pkgSubIDs := strings.Split(pkg.ID, " ")
		fmt.Printf("List Defines Package: %s\n", pkgSubIDs[0])

		// List files
		for _, vFile := range pkg.Syntax {
			filePosition := app.Fset.Position(vFile.Package)
			fileName := filePosition.Filename

			if _, ok := FileChanges[fileName]; ok {
				continue
			}

			fc := &FileChanging{
				FileName:      fileName,
				AstFile:       vFile,
				Pkg:           pkg,
				ImportNames:   make(map[string]string),
				SkipAddImport: false,
				Count:         0,
			}
			fc.ImportNames[app.PkgTo] = app.PkgToName
			FileChanges[fileName] = fc
		}

		// Imports...
		for _, obj := range pkg.TypesInfo.Uses {
			if pkgName, ok := obj.(*types.PkgName); ok {
				pkgPosition := app.Fset.Position((pkgName.Pos()))
				if fc, ok := FileChanges[pkgPosition.Filename]; ok {
					pkgPath := pkgName.Imported().Path()
					if pkgPath == app.PkgTo {
						fc.SkipAddImport = true
					}
					fc.ImportNames[pkgPath] = pkgName.Name()
				}
			}
		}

		// Tyoes defs ...
		for _, obj := range pkg.TypesInfo.Defs {
			if typeName, ok := obj.(*types.TypeName); ok {
				pkgPath := typeName.Pkg().Path()
				if pkgPath == app.PkgFrom {
					whereDefined := app.Fset.Position(typeName.Pos())
					if strings.Contains(whereDefined.Filename, app.NamesLocated) {
						app.NamesFrom[typeName.Name()] = whereDefined.Filename
						fmt.Printf("Add Name %s %s\n", typeName.Name(), whereDefined.Filename)
					} else {
						fmt.Printf("Skip Name %s %s\n", typeName.Name(), whereDefined.Filename)
					}
				}
			}
		}
	}
}

func (app *App) Apply() {
	for _, fc := range FileChanges {
		fmt.Printf("Try Apply Filename: %s\n", fc.FileName)

		astutil.Apply(fc.AstFile, func(cr *astutil.Cursor) bool {
			return app.MyApplier(cr, fc)
		}, nil)

		if fc.Count > 0 {
			if app.IsDryRun {
				fmt.Printf("Need Changing Filename: %s\n", fc.FileName)
			}

			if !fc.SkipAddImport {
				astutil.AddNamedImport(app.Fset, fc.AstFile, app.PkgToName, app.PkgTo)
				fmt.Printf("Add Import: %s %s\n", app.PkgToName, app.PkgTo)
			}

			if !app.IsDryRun {
				err := app.WriteFile(fc)
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

func (app *App) WriteFile(fc *FileChanging) error {
	fmt.Printf("Writing Filename: %s\n", fc.FileName)
	// fmt.Println(buf.String())

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, app.Fset, fc.AstFile); err != nil {
		return err
	}

	f, err := os.Create(fc.FileName)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(buf.Bytes())
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}

	return nil
}

func main() {

	app := NewApp()
	if app == nil {
		os.Exit(1)
		return
	}

	app.LoadPackages()
	app.ListDefines()
	app.Apply()
}
