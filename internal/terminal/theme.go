package terminal

// Theme holds ANSI color codes for each visual element.
type Theme struct {
	H1Color       string
	H2Color       string
	H3Color       string
	H4Color       string
	H5Color       string
	H6Color       string
	CodeBg        string
	CodeBorder    string
	BlockquoteBar string
	LinkColor     string
	TableBorder   string
	TableHeader   string
	HRColor       string
	ImageCaption  string
	ErrorColor    string
}

// DefaultTheme returns the default TrueColor theme.
func DefaultTheme() *Theme {
	return &Theme{
		H1Color:       "\033[1;36m", // bold cyan
		H2Color:       "\033[1;32m", // bold green
		H3Color:       "\033[1;33m", // bold yellow
		H4Color:       "\033[1;34m", // bold blue
		H5Color:       "\033[1;35m", // bold magenta
		H6Color:       "\033[1;37m", // bold white
		CodeBg:        "\033[48;5;236m",
		CodeBorder:    "\033[38;5;240m",
		BlockquoteBar: "\033[33m", // yellow
		LinkColor:     "\033[4;36m",
		TableBorder:   "\033[38;5;240m",
		TableHeader:   "\033[1;37m",
		HRColor:       "\033[38;5;240m",
		ImageCaption:  "\033[3;36m", // italic cyan
		ErrorColor:    "\033[1;31m", // bold red
	}
}
