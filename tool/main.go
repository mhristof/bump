package tool

type Change struct {
	Repo       string
	Version    string
	OldVersion string
}

type Update interface {
	Update(string, bool) ([]Change, error)
}
