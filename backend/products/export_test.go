package products

import "net/http"

// SetPhotoClient lets tests download photos from a test server.
func SetPhotoClient(c *http.Client) { photoClient = c }
