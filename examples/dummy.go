package examples

// DDBai ...
type DDBai struct {
	Name string
}

// DReturnCategory ...
func (st *DServiceType) DReturnCategory() *DServiceCategory {
	zs := DServiceCategory{
		ID: 2021,
		DServiceType: &DServiceType{
			Username: "Udin",
			DServiceBase: &DServiceBase {
				Phonenumber: "6522221111",
			},
		},
	}

	zs.DServiceType.Username = "Rudi"

	return &DServiceCategory{
		ID: 20,
	}
}
