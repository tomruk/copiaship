// Code generated by 'yaegi extract github.com/gen2brain/beeep'. DO NOT EDIT.

package symbols

import (
	"github.com/gen2brain/beeep"
	"reflect"
)

func init() {
	Symbols["github.com/gen2brain/beeep/beeep"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"Alert":           reflect.ValueOf(beeep.Alert),
		"Beep":            reflect.ValueOf(beeep.Beep),
		"DefaultDuration": reflect.ValueOf(&beeep.DefaultDuration).Elem(),
		"DefaultFreq":     reflect.ValueOf(&beeep.DefaultFreq).Elem(),
		"ErrUnsupported":  reflect.ValueOf(&beeep.ErrUnsupported).Elem(),
		"Notify":          reflect.ValueOf(beeep.Notify),
	}
}
