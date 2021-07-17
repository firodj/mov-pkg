package examples

// DServiceType ...
type DServiceType struct {
	ID       int `json:"-" faker:"-"`
	Username string `json:"username"`
}

// DServiceCategory...
type DServiceCategory struct {
	ID            int           `json:"-"`
	DServiceType  *DServiceType `json:"service_type,omitempty" pg:"rel:has-one"`
	DServiceType2 DServiceType  `json:"service_type2,omitempty" pg:"rel:has-one"`
}
