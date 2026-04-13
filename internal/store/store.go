package servicestore

import "gateway/internal/store/manager"

type Storer interface {
	Store()
	List()
	Get(manager.Protector, manager.Observer) manager.ServiceManger
}
