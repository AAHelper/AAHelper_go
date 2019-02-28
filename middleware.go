package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bluele/gcache"

	"github.com/flosch/pongo2"
	. "github.com/gin-gonic/gin"
)

var templates = make(map[string]*pongo2.Template)
var gc gcache.Cache

func init() {
	gc = gcache.New(20).
		ARC().
		Expiration(time.Minute * 5).
		Build()
}

func getCompiledTemplate(templateName string) *pongo2.Template {
	template, exists := templates[templateName]
	// pretty.Println("Template: " + templateName + " Exists?" + strconv.FormatBool(exists))
	if exists {
		return template
	}
	template = pongo2.Must(pongo2.FromFile(templateName))
	templates[templateName] = template
	return template
}

//Pongo2 handler for templates
func Pongo2() HandlerFunc {
	return func(c *Context) {
		c.Next()

		name, err := stringFromContext(c, "template")
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		}

		data, _ := c.Get("data")

		// template := pongo2.Must(pongo2.FromFile(name))
		template := getCompiledTemplate(name)
		// var buf bytes.Buffer

		err = template.ExecuteWriter(convertContext(data), c.Writer)
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		}

		// c.Writer.WriteString(buf)
		// if err != nil {
		// 	http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		// }
	}
}

func stringFromContext(c *Context, input string) (string, error) {
	raw, ok := c.Get(input)
	if ok {
		strVal, ok := raw.(string)
		if ok {
			return strVal, nil
		}
	}
	return "", fmt.Errorf("No data for context variable: %s", input)
}

func convertContext(thing interface{}) pongo2.Context {
	if thing != nil {
		context, isMap := thing.(map[string]interface{})
		if isMap {
			return context
		}
	}
	return nil
}

func getContext(templateData interface{}, err error) pongo2.Context {
	if templateData == nil || err != nil {
		return nil
	}
	contextData, isMap := templateData.(map[string]interface{})
	if isMap {
		return contextData
	}
	return nil
}
