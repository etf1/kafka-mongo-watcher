package config

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"text/tabwriter"
)

func TableString(iface interface{}) string {
	b := &bytes.Buffer{}
	w := tabwriter.NewWriter(b, 0, 0, 1, ' ', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprint(w, "\n-----------------------------------\n")
	fprint(w, iface)
	fmt.Fprint(w, "-----------------------------------\n")
	w.Flush()

	return b.String()
}

func fprint(w io.Writer, iface interface{}) {
	value := reflect.ValueOf(iface)

	if value.Kind() != reflect.Struct {
		value = value.Elem()
	}

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if !field.CanInterface() {
			continue
		}
		typeField := value.Type().Field(i)
		if field.Kind() == reflect.Struct {
			fmt.Fprintf(w, "##### %s #####\n", typeField.Name)
			iface := field.Interface()
			fprint(w, iface)

			continue
		}

		val := field.Interface()
		if v, ok := typeField.Tag.Lookup("print"); ok && v == "-" {
			val = "*** Hidden value ***"
		}
		fmt.Fprintf(w, "%s\t\x1b[0m%v\t\x1b[1;34m%s\x1b[0m \x1b[1;92m`%s`\x1b[0m\n", typeField.Name, val, field.Type().String(), typeField.Tag)
	}
}
