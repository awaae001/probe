package handlers

// Handlers is a slice of all available event handlers.
// Add new handlers to this slice to have them automatically registered.
var Handlers = []interface{}{
	MessageCreate,
	ThreadCreateHandler,
	ThreadDeleteHandler,
}
