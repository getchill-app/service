package service

type errNoCertFound struct{}

func (e errNoCertFound) Error() string {
	return "certificate not found"
}
