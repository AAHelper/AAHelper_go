package gforms

// Generate text input fiele: <input type="time" ...>
func TimeInputWidget(attrs map[string]string) Widget {
	w := new(textInputWidget)
	w.Type = "time"
	if attrs == nil {
		attrs = map[string]string{}
	}
	w.Attrs = attrs
	return w
}

