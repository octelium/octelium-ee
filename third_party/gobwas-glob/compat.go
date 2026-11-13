package glob

type Glob interface {
	Match(string) bool
}
