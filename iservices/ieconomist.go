package iservices

var ECO_SERVICE_NAME = "economist"

type IEconomist interface {
	Do() error
}
