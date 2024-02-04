package postgres

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm/schema"
)

/*
	This serializer is used to convert a time.Duration to a postgres int field
	representing the number of seconds in the duration.
	For example, 1h30m would be stored as 5400.
*/

type TimeDurationSecondsSerializer struct{}

func (TimeDurationSecondsSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) (err error) {
	dbValueIsNil := dbValue == nil

	structFieldType := field.StructField.Type
	structFieldIsPointer := structFieldType.Kind() == reflect.Ptr

	if !structFieldIsPointer && dbValueIsNil {
		return fmt.Errorf("can't scan nil value to non-pointer target %s", structFieldType)
	}

	if structFieldIsPointer && dbValueIsNil {
		field.ReflectValueOf(ctx, dst).Set(reflect.Zero(structFieldType))

		return nil
	}

	seconds, ok := dbValue.(int64)
	if !ok {
		return errors.Errorf("expected int64, got %T", dbValue)
	}
	dur := time.Duration(seconds) * time.Second

	var v reflect.Value
	if structFieldIsPointer {
		v = reflect.ValueOf(&dur)
	} else {
		v = reflect.ValueOf(dur)
	}

	field.ReflectValueOf(ctx, dst).Set(v)

	return nil
}

func (TimeDurationSecondsSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	if reflect.TypeOf(fieldValue).Kind() == reflect.Ptr {
		if reflect.ValueOf(fieldValue).IsNil() {
			return nil, nil
		}

		fieldValue = reflect.ValueOf(fieldValue).Elem().Interface()
	}

	dur, ok := fieldValue.(time.Duration)
	if !ok {
		return nil, errors.Errorf("expected time.Duration, got %T", fieldValue)
	}

	secs := int64(dur / time.Second)

	return secs, nil
}

func init() {
	schema.RegisterSerializer("timeDurationSeconds", &TimeDurationSecondsSerializer{})
}
