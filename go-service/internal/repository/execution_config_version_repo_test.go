package repository

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
)

func TestBindingVersionResultReturnsNilWhenBindingDoesNotExist(t *testing.T) {
	version, err := bindingVersionResult(model.ExecutionConfigVersion{}, gorm.ErrRecordNotFound)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("应保留未找到错误，实际错误: %v", err)
	}
	if version != nil {
		t.Fatalf("未找到绑定时必须返回 nil，不能返回全零版本: %#v", version)
	}
}

func TestBindingVersionResultReturnsVersionOnSuccess(t *testing.T) {
	want := model.ExecutionConfigVersion{ID: uuid.New(), VersionNo: 3}
	got, err := bindingVersionResult(want, nil)
	if err != nil {
		t.Fatalf("读取成功不应返回错误: %v", err)
	}
	if got == nil || got.ID != want.ID || got.VersionNo != want.VersionNo {
		t.Fatalf("返回版本不正确: got=%#v want=%#v", got, want)
	}
}
