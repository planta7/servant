// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package server

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const fileServerCss = `
	<style>
		* {
			color: darkslategray;
		}

		body {
			font-family: Trebuchet MS;
			padding: 20px;
		}

		body > a:first-child {
			display: block;
			margin-bottom: 40px;
		}

		table {
			border-collapse: collapse;
		}

		tr:first-child td {
			font-weight: 600;
			font-family: inherit;
			border-bottom: 10px solid transparent;
			padding-bottom: 10px;
		}

		tr td:last-child {
			padding-left: 80px;
		}

		tr {
			height: 30px;
			border-bottom: 1px solid gainsboro;
		}

		tr a[href$="/"] {
			font-weight: 600;
		}

		td:nth-child(1), td:nth-child(2) {  
			font-family: 'Courier New';
		}

		td:nth-child(1) {
			width: 35px;
		}

		td:nth-child(2) {
			width: 600px;
		}

		td:nth-child(3) {
			width: 100px;
			text-align: right;
		}

		td:nth-child(4) {
			width: 290px;
		}
	</style>
	`

// The algorithm uses at most sniffLen bytes to make its decision.
const sniffLen = 512

// condResult is the result of an HTTP request precondition check.
// See https://tools.ietf.org/html/rfc7232 section 3.
type condResult int

const (
	condNone condResult = iota
	condTrue
	condFalse
)

type anyDirs interface {
	len() int
	name(i int) string
	size(i int) int64
	lastModified(i int) time.Time
	isDir(i int) bool
	fType(i int) string
}

type fileInfoDirs []fs.FileInfo

func (d fileInfoDirs) len() int                     { return len(d) }
func (d fileInfoDirs) isDir(i int) bool             { return d[i].IsDir() }
func (d fileInfoDirs) name(i int) string            { return d[i].Name() }
func (d fileInfoDirs) size(i int) int64             { return d[i].Size() }
func (d fileInfoDirs) lastModified(i int) time.Time { return d[i].ModTime() }
func (d fileInfoDirs) fType(i int) string {
	if d[i].IsDir() {
		return "D"
	}
	return "F"
}

type dirEntryDirs []fs.DirEntry

func (d dirEntryDirs) len() int          { return len(d) }
func (d dirEntryDirs) isDir(i int) bool  { return d[i].IsDir() }
func (d dirEntryDirs) name(i int) string { return d[i].Name() }
func (d dirEntryDirs) lastModified(i int) time.Time {
	info, _ := d[i].Info()
	return info.ModTime()
}

func (d dirEntryDirs) size(i int) int64 {
	info, _ := d[i].Info()
	return info.Size()
}

func (d dirEntryDirs) fType(i int) string {
	if d[i].IsDir() {
		return "D"
	}
	return "F"
}

type fileHandler struct {
	root http.FileSystem
}

type Handler interface {
	http.Handler
}

func FileServer(root http.FileSystem) Handler {
	return &fileHandler{root}
}

func (f *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}
	serveFile(w, r, f.root, path.Clean(upath), true)
}

// name is '/'-separated, not filepath.Separator.
func serveFile(w http.ResponseWriter, r *http.Request, fs http.FileSystem, name string, redirect bool) {
	const indexPage = "/index.html"

	// redirect .../index.html to .../
	// can't use Redirect() because that would make the path absolute,
	// which would be a problem running under StripPrefix
	if strings.HasSuffix(r.URL.Path, indexPage) {
		localRedirect(w, r, "./")
		return
	}

	f, err := fs.Open(name)
	if err != nil {
		msg, code := toHTTPError(err)
		http.Error(w, msg, code)
		return
	}
	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		msg, code := toHTTPError(err)
		http.Error(w, msg, code)
		return
	}

	if redirect {
		// redirect to canonical path: / at end of directory url
		// r.URL.Path always begins with /
		_url := r.URL.Path
		if d.IsDir() {
			if _url[len(_url)-1] != '/' {
				localRedirect(w, r, path.Base(_url)+"/")
				return
			}
		} else {
			if _url[len(_url)-1] == '/' {
				localRedirect(w, r, "../"+path.Base(_url))
				return
			}
		}
	}

	if d.IsDir() {
		_url := r.URL.Path
		// redirect if the directory name doesn't end in a slash
		if _url == "" || _url[len(_url)-1] != '/' {
			localRedirect(w, r, path.Base(_url)+"/")
			return
		}

		// use contents of index.html for directory, if present
		index := strings.TrimSuffix(name, "/") + indexPage
		ff, err := fs.Open(index)
		if err == nil {
			defer ff.Close()
			dd, err := ff.Stat()
			if err == nil {
				d = dd
				f = ff
			}
		}
	}

	// Still a directory? (we didn't find an index.html file)
	if d.IsDir() {
		if checkIfModifiedSince(r, d.ModTime()) == condFalse {
			writeNotModified(w)
			return
		}
		setLastModified(w, d.ModTime())
		dirList(w, r, f)
		return
	}

	// serveContent will check modification time
	sizeFunc := func() (int64, error) { return d.Size(), nil }
	serveContent(w, r, d.Name(), d.ModTime(), sizeFunc, f)
}

// localRedirect gives a Moved Permanently response.
// It does not convert relative paths to absolute paths like Redirect does.
func localRedirect(w http.ResponseWriter, r *http.Request, newPath string) {
	if q := r.URL.RawQuery; q != "" {
		newPath += "?" + q
	}
	w.Header().Set("Location", newPath)
	w.WriteHeader(http.StatusMovedPermanently)
}

// toHTTPError returns a non-specific HTTP error message and status code
// for a given non-nil error value. It's important that toHTTPError does not
// actually return err.Error(), since msg and httpStatus are returned to users,
// and historically Go's ServeContent always returned just "404 Not Found" for
// all errors. We don't want to start leaking information in error messages.
func toHTTPError(err error) (msg string, httpStatus int) {
	if errors.Is(err, fs.ErrNotExist) {
		return "404 page not found", http.StatusNotFound
	}
	if errors.Is(err, fs.ErrPermission) {
		return "403 Forbidden", http.StatusForbidden
	}
	// Default:
	return "500 Internal Server Error", http.StatusInternalServerError
}

func checkIfRange(w http.ResponseWriter, r *http.Request, modtime time.Time) condResult {
	if r.Method != "GET" && r.Method != "HEAD" {
		return condNone
	}
	ir := r.Header.Get("If-Range")
	if ir == "" {
		return condNone
	}
	etag, _ := scanETag(ir)
	if etag != "" {
		if etagStrongMatch(etag, w.Header().Get("Etag")) {
			return condTrue
		} else {
			return condFalse
		}
	}
	// The If-Range value is typically the ETag value, but it may also be
	// the modtime date. See golang.org/issue/8367.
	if modtime.IsZero() {
		return condFalse
	}
	t, err := http.ParseTime(ir)
	if err != nil {
		return condFalse
	}
	if t.Unix() == modtime.Unix() {
		return condTrue
	}
	return condFalse
}

func checkIfModifiedSince(r *http.Request, modtime time.Time) condResult {
	if r.Method != "GET" && r.Method != "HEAD" {
		return condNone
	}
	ims := r.Header.Get("If-Modified-Since")
	if ims == "" || isZeroTime(modtime) {
		return condNone
	}
	t, err := http.ParseTime(ims)
	if err != nil {
		return condNone
	}
	// The Last-Modified header truncates sub-second precision so
	// the modtime needs to be truncated too.
	modtime = modtime.Truncate(time.Second)
	if ret := modtime.Compare(t); ret <= 0 {
		return condFalse
	}
	return condTrue
}

var unixEpochTime = time.Unix(0, 0)

// isZeroTime reports whether t is obviously unspecified (either zero or Unix()=0).
func isZeroTime(t time.Time) bool {
	return t.IsZero() || t.Equal(unixEpochTime)
}

func setLastModified(w http.ResponseWriter, modtime time.Time) {
	if !isZeroTime(modtime) {
		w.Header().Set("Last-Modified", modtime.UTC().Format(http.TimeFormat))
	}
}

func writeNotModified(w http.ResponseWriter) {
	// RFC 7232 section 4.1:
	// a sender SHOULD NOT generate representation metadata other than the
	// above listed fields unless said metadata exists for the purpose of
	// guiding cache updates (e.g., Last-Modified might be useful if the
	// response does not have an ETag field).
	h := w.Header()
	delete(h, "Content-Type")
	delete(h, "Content-Length")
	delete(h, "Content-Encoding")
	if h.Get("Etag") != "" {
		delete(h, "Last-Modified")
	}
	w.WriteHeader(http.StatusNotModified)
}

var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	// "&#34;" is shorter than "&quot;".
	`"`, "&#34;",
	// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	"'", "&#39;",
)

func wrapFprintf(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}

func dirList(w http.ResponseWriter, r *http.Request, f http.File) {
	// Prefer to use ReadDir instead of Readdir,
	// because the former doesn't require calling
	// Stat on every entry of a directory on Unix.
	var dirs anyDirs
	var err error
	if d, ok := f.(fs.ReadDirFile); ok {
		var list dirEntryDirs
		list, err = d.ReadDir(-1)
		dirs = list
	} else {
		var list fileInfoDirs
		list, err = f.Readdir(-1)
		dirs = list
	}

	if err != nil {
		logf(r, "http: error reading directory: %v", err)
		http.Error(w, "Error reading directory", http.StatusInternalServerError)
		return
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs.name(i) < dirs.name(j) })
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	info, _ := f.Stat()
	if info.Name() != "." {
		wrapFprintf(w, "<a href=\"%s\">%s</a>\n", "..", "↑ Parent directory")
	}
	wrapFprintf(w, "<table>\n")
	if dirs.len() == 0 {
		wrapFprintf(w, "<h4>Directory is empty</h4>")
	} else {
		_, _ = fmt.Fprint(w, "<tr><td/><td>Name</a><td>Size</td><td>Last modified</td></tr>\n")
	}
	for i, n := 0, dirs.len(); i < n; i++ {
		name := dirs.name(i)
		size := dirs.size(i)
		lastMod := dirs.lastModified(i)
		fType := dirs.fType(i)
		if dirs.isDir(i) {
			name += "/"
		}
		// name may contain '?' or '#', which must be escaped to remain
		// part of the URL path, and not indicate the start of a query
		// string or fragment.
		_url := url.URL{Path: name}
		wrapFprintf(w, "<tr><td>[%s]</td><td><a href=\"%s\">%s</a><td>%s</td><td>%s</td></tr>\n",
			fType,
			_url.String(),
			htmlReplacer.Replace(name),
			humanizeSize(size),
			lastMod.Format(time.RFC1123Z),
		)
	}
	wrapFprintf(w, "</table>\n")
	wrapFprintf(w, fileServerCss)
}

// if name is empty, filename is unknown. (used for mime type, before sniffing)
// if modtime.IsZero(), modtime is unknown.
// content must be seeked to the beginning of the file.
// The sizeFunc is called at most once. Its error, if any, is sent in the HTTP response.
func serveContent(w http.ResponseWriter, r *http.Request, name string, modtime time.Time, sizeFunc func() (int64, error), content io.ReadSeeker) {
	setLastModified(w, modtime)
	done, rangeReq := checkPreconditions(w, r, modtime)
	if done {
		return
	}

	code := http.StatusOK

	// If Content-Type isn't set, use the file's extension to find it, but
	// if the Content-Type is unset explicitly, do not sniff the type.
	ctypes, haveType := w.Header()["Content-Type"]
	var ctype string
	if !haveType {
		ctype = mime.TypeByExtension(filepath.Ext(name))
		if ctype == "" {
			// read a chunk to decide between utf-8 text and binary
			var buf [sniffLen]byte
			n, _ := io.ReadFull(content, buf[:])
			ctype = http.DetectContentType(buf[:n])
			_, err := content.Seek(0, io.SeekStart) // rewind to output whole file
			if err != nil {
				http.Error(w, "seeker can't seek", http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Content-Type", ctype)
	} else if len(ctypes) > 0 {
		ctype = ctypes[0]
	}

	size, err := sizeFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if size < 0 {
		// Should never happen but just to be sure
		http.Error(w, "negative content size computed", http.StatusInternalServerError)
		return
	}

	// handle Content-Range header.
	sendSize := size
	var sendContent io.Reader = content
	ranges, err := parseRange(rangeReq, size)
	switch err {
	case nil:
	case errNoOverlap:
		if size == 0 {
			// Some clients add a Range header to all requests to
			// limit the size of the response. If the file is empty,
			// ignore the range header and respond with a 200 rather
			// than a 416.
			ranges = nil
			break
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
		fallthrough
	default:
		http.Error(w, err.Error(), http.StatusRequestedRangeNotSatisfiable)
		return
	}

	if sumRangesSize(ranges) > size {
		// The total number of bytes in all the ranges
		// is larger than the size of the file by
		// itself, so this is probably an attack, or a
		// dumb client. Ignore the range request.
		ranges = nil
	}
	switch {
	case len(ranges) == 1:
		// RFC 7233, Section 4.1:
		// "If a single part is being transferred, the server
		// generating the 206 response MUST generate a
		// Content-Range header field, describing what range
		// of the selected representation is enclosed, and a
		// payload consisting of the range.
		// ...
		// A server MUST NOT generate a multipart response to
		// a request for a single range, since a client that
		// does not request multiple parts might not support
		// multipart responses."
		ra := ranges[0]
		if _, err := content.Seek(ra.start, io.SeekStart); err != nil {
			http.Error(w, err.Error(), http.StatusRequestedRangeNotSatisfiable)
			return
		}
		sendSize = ra.length
		code = http.StatusPartialContent
		w.Header().Set("Content-Range", ra.contentRange(size))
	case len(ranges) > 1:
		sendSize = rangesMIMESize(ranges, ctype, size)
		code = http.StatusPartialContent

		pr, pw := io.Pipe()
		mw := multipart.NewWriter(pw)
		w.Header().Set("Content-Type", "multipart/byteranges; boundary="+mw.Boundary())
		sendContent = pr
		defer pr.Close() // cause writing goroutine to fail and exit if CopyN doesn't finish.
		go func() {
			for _, ra := range ranges {
				part, err := mw.CreatePart(ra.mimeHeader(ctype, size))
				if err != nil {
					_ = pw.CloseWithError(err)
					return
				}
				if _, err := content.Seek(ra.start, io.SeekStart); err != nil {
					_ = pw.CloseWithError(err)
					return
				}
				if _, err := io.CopyN(part, content, ra.length); err != nil {
					_ = pw.CloseWithError(err)
					return
				}
			}
			_ = mw.Close()
			_ = pw.Close()
		}()
	}

	w.Header().Set("Accept-Ranges", "bytes")
	if w.Header().Get("Content-Encoding") == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(sendSize, 10))
	}

	w.WriteHeader(code)

	if r.Method != "HEAD" {
		_, _ = io.CopyN(w, sendContent, sendSize)
	}
}

// logf prints to the ErrorLog of the *Server associated with request r
// via ServerContextKey. If there's no associated server, or if ErrorLog
// is nil, logging is done via the log package's standard logger.
func logf(r *http.Request, format string, args ...any) {
	s, _ := r.Context().Value(http.ServerContextKey).(*http.Server)
	if s != nil && s.ErrorLog != nil {
		s.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

// checkPreconditions evaluates request preconditions and reports whether a precondition
// resulted in sending StatusNotModified or StatusPreconditionFailed.
func checkPreconditions(w http.ResponseWriter, r *http.Request, modtime time.Time) (done bool, rangeHeader string) {
	// This function carefully follows RFC 7232 section 6.
	ch := checkIfMatch(w, r)
	if ch == condNone {
		ch = checkIfUnmodifiedSince(r, modtime)
	}
	if ch == condFalse {
		w.WriteHeader(http.StatusPreconditionFailed)
		return true, ""
	}
	switch checkIfNoneMatch(w, r) {
	case condFalse:
		if r.Method == "GET" || r.Method == "HEAD" {
			writeNotModified(w)
			return true, ""
		} else {
			w.WriteHeader(http.StatusPreconditionFailed)
			return true, ""
		}
	case condNone:
		if checkIfModifiedSince(r, modtime) == condFalse {
			writeNotModified(w)
			return true, ""
		}
	}

	rangeHeader = r.Header.Get("Range")
	if rangeHeader != "" && checkIfRange(w, r, modtime) == condFalse {
		rangeHeader = ""
	}
	return false, rangeHeader
}

func checkIfMatch(w http.ResponseWriter, r *http.Request) condResult {
	im := r.Header.Get("If-Match")
	if im == "" {
		return condNone
	}
	for {
		im = textproto.TrimString(im)
		if len(im) == 0 {
			break
		}
		if im[0] == ',' {
			im = im[1:]
			continue
		}
		if im[0] == '*' {
			return condTrue
		}
		etag, remain := scanETag(im)
		if etag == "" {
			break
		}
		if etagStrongMatch(etag, w.Header().Get("Etag")) {
			return condTrue
		}
		im = remain
	}

	return condFalse
}

func checkIfUnmodifiedSince(r *http.Request, modtime time.Time) condResult {
	ius := r.Header.Get("If-Unmodified-Since")
	if ius == "" || isZeroTime(modtime) {
		return condNone
	}
	t, err := http.ParseTime(ius)
	if err != nil {
		return condNone
	}

	// The Last-Modified header truncates sub-second precision so
	// the modtime needs to be truncated too.
	modtime = modtime.Truncate(time.Second)
	if ret := modtime.Compare(t); ret <= 0 {
		return condTrue
	}
	return condFalse
}

func checkIfNoneMatch(w http.ResponseWriter, r *http.Request) condResult {
	inm := r.Header.Get("If-None-Match")
	if inm == "" {
		return condNone
	}
	buf := inm
	for {
		buf = textproto.TrimString(buf)
		if len(buf) == 0 {
			break
		}
		if buf[0] == ',' {
			buf = buf[1:]
			continue
		}
		if buf[0] == '*' {
			return condFalse
		}
		etag, remain := scanETag(buf)
		if etag == "" {
			break
		}
		if etagWeakMatch(etag, w.Header().Get("Etag")) {
			return condFalse
		}
		buf = remain
	}
	return condTrue
}

// httpRange specifies the byte range to be sent to the client.
type httpRange struct {
	start, length int64
}

func (r httpRange) contentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.start, r.start+r.length-1, size)
}

func (r httpRange) mimeHeader(contentType string, size int64) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Range": {r.contentRange(size)},
		"Content-Type":  {contentType},
	}
}

// errNoOverlap is returned by serveContent's parseRange if first-byte-pos of
// all of the byte-range-spec values is greater than the content size.
var errNoOverlap = errors.New("invalid range: failed to overlap")

// parseRange parses a Range header string as per RFC 7233.
// errNoOverlap is returned if none of the ranges overlap.
func parseRange(s string, size int64) ([]httpRange, error) {
	if s == "" {
		return nil, nil // header not present
	}
	const b = "bytes="
	if !strings.HasPrefix(s, b) {
		return nil, errors.New("invalid range")
	}
	var ranges []httpRange
	noOverlap := false
	for _, ra := range strings.Split(s[len(b):], ",") {
		ra = textproto.TrimString(ra)
		if ra == "" {
			continue
		}
		start, end, ok := strings.Cut(ra, "-")
		if !ok {
			return nil, errors.New("invalid range")
		}
		start, end = textproto.TrimString(start), textproto.TrimString(end)
		var r httpRange
		if start == "" {
			// If no start is specified, end specifies the
			// range start relative to the end of the file,
			// and we are dealing with <suffix-length>
			// which has to be a non-negative integer as per
			// RFC 7233 Section 2.1 "Byte-Ranges".
			if end == "" || end[0] == '-' {
				return nil, errors.New("invalid range")
			}
			i, err := strconv.ParseInt(end, 10, 64)
			if i < 0 || err != nil {
				return nil, errors.New("invalid range")
			}
			if i > size {
				i = size
			}
			r.start = size - i
			r.length = size - r.start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				return nil, errors.New("invalid range")
			}
			if i >= size {
				// If the range begins after the size of the content,
				// then it does not overlap.
				noOverlap = true
				continue
			}
			r.start = i
			if end == "" {
				// If no end is specified, range extends to end of the file.
				r.length = size - r.start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.start > i {
					return nil, errors.New("invalid range")
				}
				if i >= size {
					i = size - 1
				}
				r.length = i - r.start + 1
			}
		}
		ranges = append(ranges, r)
	}
	if noOverlap && len(ranges) == 0 {
		// The specified ranges did not overlap with the content.
		return nil, errNoOverlap
	}
	return ranges, nil
}

// scanETag determines if a syntactically valid ETag is present at s. If so,
// the ETag and remaining text after consuming ETag is returned. Otherwise,
// it returns "", "".
func scanETag(s string) (etag string, remain string) {
	s = textproto.TrimString(s)
	start := 0
	if strings.HasPrefix(s, "W/") {
		start = 2
	}
	if len(s[start:]) < 2 || s[start] != '"' {
		return "", ""
	}
	// ETag is either W/"text" or "text".
	// See RFC 7232 2.3.
	for i := start + 1; i < len(s); i++ {
		c := s[i]
		switch {
		// Character values allowed in ETags.
		case c == 0x21 || c >= 0x23 && c <= 0x7E || c >= 0x80:
		case c == '"':
			return s[:i+1], s[i+1:]
		default:
			return "", ""
		}
	}
	return "", ""
}

// etagStrongMatch reports whether a and b match using strong ETag comparison.
// Assumes a and b are valid ETags.
func etagStrongMatch(a, b string) bool {
	return a == b && a != "" && a[0] == '"'
}

// etagWeakMatch reports whether a and b match using weak ETag comparison.
// Assumes a and b are valid ETags.
func etagWeakMatch(a, b string) bool {
	return strings.TrimPrefix(a, "W/") == strings.TrimPrefix(b, "W/")
}

// countingWriter counts how many bytes have been written to it.
type countingWriter int64

func (w *countingWriter) Write(p []byte) (n int, err error) {
	*w += countingWriter(len(p))
	return len(p), nil
}

// rangesMIMESize returns the number of bytes it takes to encode the
// provided ranges as a multipart response.
func rangesMIMESize(ranges []httpRange, contentType string, contentSize int64) (encSize int64) {
	var w countingWriter
	mw := multipart.NewWriter(&w)
	for _, ra := range ranges {
		_, _ = mw.CreatePart(ra.mimeHeader(contentType, contentSize))
		encSize += ra.length
	}
	_ = mw.Close()
	encSize += int64(w)
	return
}

func sumRangesSize(ranges []httpRange) (size int64) {
	for _, ra := range ranges {
		size += ra.length
	}
	return
}

func humanizeSize(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
