package main

type Route struct {
	Method      string            // HTTP method (GET, POST, etc.)
	Path        string            // URL path pattern
	Handler     string            // Name of the handler function
	Description string            // Description from comments
	BodyType    string            // Name of struct to use for body
	Tags        map[string]string // Any route tags
	RequestBody *RequestBody      // Request body information
}

type RequestBody struct {
	TypeName    string
	Fields      []RequestBodyField
	Description string
}

type RequestBodyField struct {
	Name        string
	Type        string
	JSONName    string
	Required    bool
	Description string
	Tags        map[string]string
}
