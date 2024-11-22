package sdkforms

type JsonSection struct {
	Name   string
	Fields []JsonField
}

type JsonField struct {
	Name         string
	Label        string
	Type         string
	ListMultiple bool
	ListOptions  []JsonListOpt
	MultiColumns []JsonMultiCol
	Value        interface{}
}

type JsonMultiCol struct {
	Name  string
	Label string
	Type  string
}

type JsonListOpt struct {
	Label    string
	Value    string
	Selected bool
}
