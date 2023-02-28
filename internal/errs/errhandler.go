package errs

import (
	"fmt"
)

type ErrWithDescription interface {
	Error() string
	GetDescription() string
}

type ErrCacheUserPolling struct {
	UserID            int64
	RequisitionNumber int64
}

func (e *ErrCacheUserPolling) Error() string {
	return fmt.Sprintf("Cache userPolling for user (userID = %v) is full.", e.UserID)
}

func (e *ErrCacheUserPolling) GetDescription() string {
	return fmt.Sprintf("В данный момент автор заявки №%v (userID %v) работает над новой заявкой :)\n Я сообщю когда пользователь %v (автор заявки №%v) завершит работу.", e.RequisitionNumber, e.UserID, e.UserID, e.RequisitionNumber)
}

func ErrorHandler(err error) error {
	if difErr, ok := err.(*ErrCacheUserPolling); ok {
		return fmt.Errorf(difErr.GetDescription())
	}

	return err
}
