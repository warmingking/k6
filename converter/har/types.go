/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2017 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package har

import (
	"time"
)

// HAR is the top level object of a HAR log.
type HAR struct {
	Log *Log `json:"log"`
}

// Log is the HAR HTTP request and response log.
type Log struct {
	Creator *Creator `json:"creator"`
	Browser *Browser `json:"browser,omitempty"`
	Version string   `json:"version"`
	Comment string   `json:"comment,omitempty"`
	Pages   []Page   `json:"pages,omitempty"`
	Entries []*Entry `json:"entries"`
}

// Creator is the program responsible for generating the log. Martian, in this case.
type Creator struct {
	// Name of the log creator application.
	Name string `json:"name"`
	// Version of the log creator application.
	Version string `json:"version"`
}

// Browser that created the log
type Browser struct {
	// Required. The name of the browser that created the log.
	Name string `json:"name"`
	// Required. The version number of the browser that created the log.
	Version string `json:"version"`
	// Optional. A comment provided by the user or the browser.
	Comment string `json:"comment"`
}

// Page object for every exported web page and one <entry> object for every HTTP request.
// In case when an HTTP trace tool isn't able to group requests by a page,
// the <pages> object is empty and individual requests doesn't have a parent page.
type Page struct {
	/* There is one <page> object for every exported web page and one <entry>
	   object for every HTTP request. In case when an HTTP trace tool isn't able to
	   group requests by a page, the <pages> object is empty and individual
	   requests doesn't have a parent page.
	*/

	// Date and time stamp for the beginning of the page load
	// (ISO 8601 YYYY-MM-DDThh:mm:ss.sTZD, e.g. 2009-07-24T19:20:30.45+01:00).
	StartedDateTime time.Time `json:"startedDateTime"`
	// Unique identifier of a page within the . Entries use it to refer the parent page.
	ID string `json:"id"`
	// Page title.
	Title string `json:"title"`
	// (new in 1.2) A comment provided by the user or the application.
	Comment string `json:"comment,omitempty"`
}

// Entry is a individual log entry for a request or response.
type Entry struct {
	StartedDateTime time.Time `json:"startedDateTime"`
	Cache           *Cache    `json:"cache"`
	Timings         *Timings  `json:"timings"`
	Request         *Request  `json:"request"`
	Response        *Response `json:"response,omitempty"`
	Pageref         string    `json:"pageref,omitempty"`
	ID              string    `json:"_id"`
	Time            float32   `json:"time"`
}

// Request holds data about an individual HTTP request.
type Request struct {
	PostData    *PostData     `json:"postData,omitempty"`
	URL         string        `json:"url"`
	HTTPVersion string        `json:"httpVersion"`
	Comment     string        `json:"comment"`
	Method      string        `json:"method"`
	Headers     []Header      `json:"headers"`
	QueryString []QueryString `json:"queryString"`
	Cookies     []Cookie      `json:"cookies"`
	HeadersSize int64         `json:"headersSize"`
	BodySize    int64         `json:"bodySize"`
}

// Response holds data about an individual HTTP response.
type Response struct {
	Content     *Content `json:"content"`
	RedirectURL string   `json:"redirectURL"`
	StatusText  string   `json:"statusText"`
	HTTPVersion string   `json:"httpVersion"`
	Cookies     []Cookie `json:"cookies"`
	Headers     []Header `json:"headers"`
	Status      int      `json:"status"`
	HeadersSize int64    `json:"headersSize"`
	BodySize    int64    `json:"bodySize"`
}

// Cache contains information about a request coming from browser cache.
type Cache struct {
	// Has no fields as they are not supported, but HAR requires the "cache"
	// object to exist.
}

// Timings describes various phases within request-response round trip. All
// times are specified in milliseconds
type Timings struct {
	// Send is the time required to send HTTP request to the server.
	Send float32 `json:"send"`
	// Wait is the time spent waiting for a response from the server.
	Wait float32 `json:"wait"`
	// Receive is the time required to read entire response from server or cache.
	Receive float32 `json:"receive"`
}

// Cookie is the data about a cookie on a request or response.
type Cookie struct {
	// Name is the cookie name.
	Name string `json:"name"`
	// Value is the cookie value.
	Value string `json:"value"`
	// Path is the path pertaining to the cookie.
	Path string `json:"path,omitempty"`
	// Domain is the host of the cookie.
	Domain string `json:"domain,omitempty"`
	// Expires contains cookie expiration time.
	Expires time.Time `json:"-"`
	// Expires8601 contains cookie expiration time in ISO 8601 format.
	Expires8601 string `json:"expires,omitempty"`
	// HTTPOnly is set to true if the cookie is HTTP only, false otherwise.
	HTTPOnly bool `json:"httpOnly,omitempty"`
	// Secure is set to true if the cookie was transmitted over SSL, false
	// otherwise.
	Secure bool `json:"secure,omitempty"`
}

// Header is an HTTP request or response header.
type Header struct {
	// Name is the header name.
	Name string `json:"name"`
	// Value is the header value.
	Value string `json:"value"`
}

// QueryString is a query string parameter on a request.
type QueryString struct {
	// Name is the query parameter name.
	Name string `json:"name"`
	// Value is the query parameter value.
	Value string `json:"value"`
}

// PostData describes posted data on a request.
type PostData struct {
	MimeType string  `json:"mimeType"`
	Text     string  `json:"text"`
	Params   []Param `json:"params"`
}

// Param describes an individual posted parameter.
type Param struct {
	// Name of the posted parameter.
	Name string `json:"name"`
	// Value of the posted parameter.
	Value string `json:"value,omitempty"`
	// Filename of a posted file.
	Filename string `json:"fileName,omitempty"`
	// ContentType is the content type of a posted file.
	ContentType string `json:"contentType,omitempty"`
}

// Content describes details about response content.
type Content struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Size     int64  `json:"size"`
}
