package services

type Storer interface {
	Map(reqPath string) (Service, error)
}
