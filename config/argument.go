package config

type Argument struct {
	CmdType    string
	Verbose    bool
	BaseDomain string
	ApiKey     string
	Path       []string
	Type       string
	Name       string
	ExtName    string
	ShowFiles  bool
	FilePath   string
	FormatTree bool
	Overwrite  bool

	ModelVersion  []string
	VersionPublic []string
	BaseModel     []string
	CoverUrls     []string
	Intro         []string
	IntroPath     []string
	Current       int
	PageSize      int
}

func NewArgument() *Argument {
	return &Argument{}
}

func (arg *Argument) Fork() *Argument {
	args := NewArgument()
	*args = *arg
	return args
}
