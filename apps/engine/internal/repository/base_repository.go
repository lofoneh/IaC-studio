package repository

import (
	"context"
	"fmt"

	appErr "github.com/iac-studio/engine/pkg/errors"
	"gorm.io/gorm"
)

// BaseRepository defines common CRUD operations.
type BaseRepository[T any] interface {
	Create(ctx context.Context, obj *T) error
	GetByID(ctx context.Context, id any, dest *T) error
	Update(ctx context.Context, obj *T) error
	Delete(ctx context.Context, id any) error
}

type baseRepository[T any] struct {
	db *gorm.DB
}

func NewBaseRepository[T any](db *gorm.DB) BaseRepository[T] {
	return &baseRepository[T]{db: db}
}

func (r *baseRepository[T]) Create(ctx context.Context, obj *T) error {
	if err := r.db.WithContext(ctx).Create(obj).Error; err != nil {
		return appErr.Wrap(err, appErr.CodeInternal, "create entity failed")
	}
	return nil
}

func (r *baseRepository[T]) GetByID(ctx context.Context, id any, dest *T) error {
	if err := r.db.WithContext(ctx).First(dest, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return appErr.New(appErr.CodeNotFound, "entity not found")
		}
		return appErr.Wrap(err, appErr.CodeInternal, "get entity failed")
	}
	return nil
}

func (r *baseRepository[T]) Update(ctx context.Context, obj *T) error {
	if err := r.db.WithContext(ctx).Save(obj).Error; err != nil {
		return appErr.Wrap(err, appErr.CodeInternal, "update entity failed")
	}
	return nil
}

func (r *baseRepository[T]) Delete(ctx context.Context, id any) error {
	var t T
	res := r.db.WithContext(ctx).Delete(&t, "id = ?", id)
	if res.Error != nil {
		return appErr.Wrap(res.Error, appErr.CodeInternal, "delete entity failed")
	}
	if res.RowsAffected == 0 {
		return appErr.New(appErr.CodeNotFound, fmt.Sprintf("entity %v not found", id))
	}
	return nil
}


