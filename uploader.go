package letsproxy

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
)

type Uploader struct {
	TmpDir string
}

func ParseHeaderInt(w http.ResponseWriter, r *http.Request, s string, fail *bool) int {
	res, err := strconv.Atoi(r.Header.Get(s))
	if err != nil {
		log.Println("Header Error", s, err)
		//http.Error(w, "No "+s+" header.", 400)
		*fail = true
	}
	return res
}

func ParseHeaderString(w http.ResponseWriter, r *http.Request, s string, fail *bool) string {
	res := r.Header.Get(s)
	if res == "" {
		log.Println("Header Empty", s)
		//http.Error(w, "No "+s+" header.", 400)
		*fail = true
	}
	return res
}

type UploadResponse struct {
	ChunkNumber    int    `json:"chunkNumber"`
	ExpectedChunks int    `json:"expectedChunks"`
	Status         string `json:"status"`
	ChunkHash      string `json:"chunk-hash"`
	FileHash       string `json:"file-hash"`
	Debug          string `json:"debug"`
}

func (u *Uploader) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// URI: /upload?upload_uid=3d4f26c9-ea6f-457f-8678-2532e7124466&upload_path=%2Ftmp%2F
	// Headers: map[Accept-Language:[en-US,en;q=0.8] Referer:[https://code.hyperkube.co/] Chunk-Size:[768000] Chunk-Number:[1] Chunk-Hash:[cde61bc17a096bbe0702b11e75fa9fdd3421bcda] Accept-Encoding:[gzip, deflate, br] User-Agent:[Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/53.0.2785.143 Chrome/53.0.2785.143 Safari/537.36] File-Path:[keyword_hashes] Chunk-Total:[3] Accept:[*/*] Cookie:[_ga=GA1.2.52450644.1477966975; _gat=1] Content-Length:[768000] Content-Range:[bytes 0-768000/1729161] Origin:[https://code.hyperkube.co]]
	log.Println(r.URL.RequestURI())
	log.Println(r.Header)

	// parse inputs
	q := r.URL.Query()
	upload_path := "/" + q.Get("upload_path")
	uid := q.Get("upload_uid")
	fail := false
	chunk_total := ParseHeaderInt(w, r, "Chunk-Total", &fail)
	chunk_size := ParseHeaderInt(w, r, "Chunk-Size", &fail)
	chunk_number := ParseHeaderInt(w, r, "Chunk-Number", &fail)
	file_path := ParseHeaderString(w, r, "File-Path", &fail)
	chunk_hash := ParseHeaderString(w, r, "Chunk-Hash", &fail)
	file_hash := ParseHeaderString(w, r, "File-Hash", &fail)

	log.Println("Upload Path:", upload_path)
	log.Println("Upload Uid:", uid)

	log.Println("Chunk-Total", chunk_total)
	log.Println("Chunk-Size", chunk_size)
	log.Println("Chunk-Number", chunk_number)
	log.Println("File-Path", file_path)
	log.Println("Chunk-Hash", chunk_hash)
	log.Println("File-Hash", file_hash)
	log.Println("Fail", fail)

	//if fail {
	//	log.Println("Failed")
	//	return
	//}

	// main.go:152: /upload?upload_uid=af8619dd-64f9-4c22-8e31-f30af6edd478&upload_path=%2Ftmp%2F
	// main.go:153: map[Origin:[https://code.hyperkube.co] Chunk-Size:[13] Content-Range:[bytes 0-13/13] File-Hash:[cd50d19784897085a8d0e3e413f8612b097c03f1] Content-Length:[13] File-Path:[/A/B/hello_world] Chunk-Total:[1] Chunk-Hash:[cd50d19784897085a8d0e3e413f8612b097c03f1] Accept-Encoding:[gzip, deflate, br] Chunk-Number:[1]]
	// main.go:195: File Path: /A/B/hello_world
	// main.go:196: Upload Path: //tmp/
	// main.go:197: Upload Uid: af8619dd-64f9-4c22-8e31-f30af6edd478
	// main.go:198: File Hash: cd50d19784897085a8d0e3e413f8612b097c03f1

	// main.go:152: /upload?upload_uid=9aba32e7-fdf7-45d7-9442-b774ca2b21bd&upload_path=%2Ftmp%2F
	// main.go:153: map[User-Agent:[Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/53.0.2785.143 Chrome/53.0.2785.143 Safari/537.36] Content-Range:[bytes 0-5/5] Chunk-Number:[1] Chunk-Total:[1] Accept-Encoding:[gzip, deflate, br] Origin:[https://code.hyperkube.co] Accept:[*/*] Chunk-Hash:[1c68ea370b40c06fcaf7f26c8b1dba9d9caf5dea] Accept-Language:[en-US,en;q=0.8] Cookie:[_ga=GA1.2.52450644.1477966975; _gat=1] Content-Length:[5] File-Path:[/A/I have spaces] Chunk-Size:[5] File-Hash:[1c68ea370b40c06fcaf7f26c8b1dba9d9caf5dea] Referer:[https://code.hyperkube.co/]]
	// main.go:195: File Path: /A/I have spaces
	// main.go:196: Upload Path: //tmp/
	// main.go:197: Upload Uid: 9aba32e7-fdf7-45d7-9442-b774ca2b21bd
	// main.go:198: File Hash: 1c68ea370b40c06fcaf7f26c8b1dba9d9caf5dea
	enc := json.NewEncoder(w)
	response := UploadResponse{}
	// always respond with the json
	defer enc.Encode(&response)

	var chunk []byte
	var err error
	response.ChunkNumber = chunk_number
	response.ExpectedChunks = chunk_total
	chunk, err = ioutil.ReadAll(r.Body) // TODO: err?
	if err != nil {
		log.Println("Error reading chunk data")
		response.Status = "error"
		return
	}
	digest := sha1.Sum(chunk)
	hash := fmt.Sprintf("%x", digest)
	response.ChunkHash = hash
	if hash != chunk_hash {
		log.Println("Hash mismatch, server:", hash, ", client:", chunk_hash, ", chunk number:", chunk_number)
		response.Status = "failure"
		return
	}

	uip := path.Join(u.TmpDir, uid)
	file, err := os.OpenFile(uip, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	defer file.Close()
	if err != nil {
		pe := err.(*os.PathError)
		log.Println(pe.Path, pe.Err, pe.Op)
		log.Println(err)
		response.Status = "failure"
		response.Debug = err.Error()
		return
	}

	s, err := file.Write(chunk)
	if err != nil {
		log.Println(err, "wrote", s, "bytes")
		return
	}

	if chunk_number == chunk_total {
		// last chunk
		file.Close()
		// TODO: verify hash? (wtv)
		file, err = os.Open(uip)
		if err != nil {
			log.Println(err)
			// TODO
		}
		h := sha1.New()
		if _, err := io.Copy(h, file); err != nil {
			log.Println("failed to hash file", err)
		}
		my_file_hash := fmt.Sprintf("%x", h.Sum(nil))
		if my_file_hash != file_hash {
			log.Println("File Hash Mismatch", my_file_hash, file_hash)
		}
		response.FileHash = my_file_hash

		// ok, now move the file to right spot
		folder_dir := path.Dir(file_path)
		final_folder := path.Join(upload_path, folder_dir)
		err = os.MkdirAll(final_folder, os.ModePerm)
		if err != nil {
			log.Println("Error creating directory", err)
		}
		err = os.Rename(uip, path.Join(upload_path, file_path))
		if err != nil {
			log.Println("Error moving the file")
		}
	}

	// keep a map[uid]-> state of that file (not really needed, just nice)
	// write the contents in a tmp file with same name as uid
	// when you have all chunks, move the file to its final destination, creating
	// directories if needed

	response.Status = "success"
	w.WriteHeader(200)
	return
}

// Example of chunked upload:
//
// main.go:154: /upload?upload_uid=2539e718-1f39-462c-b4ce-d67f408e998a&upload_path=%2Ftmp%2F
// main.go:155: map[File-Path:[restfulobjects-spec.pdf] Content-Range:[bytes 0-768000/1963408] Chunk-Number:[1] Content-Type:[application/pdf] Origin:[https://code.hyperkube.co] User-Agent:[Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/53.0.2785.143 Chrome/53.0.2785.143 Safari/537.36] Content-Length:[768000] Cookie:[_gat=1; _ga=GA1.2.52450644.1477966975] Chunk-Size:[768000] Accept-Encoding:[gzip, deflate, br] Accept:[*/*] Referer:[https://code.hyperkube.co/] Accept-Language:[en-US,en;q=0.8] Chunk-Total:[3] Chunk-Hash:[1aa87aba497b8ead0abaafda6d1fc7fc2a095ac8]]
// main.go:135: Header Empty File-Hash
// main.go:169: Upload Path: //tmp/
// main.go:170: Upload Uid: 2539e718-1f39-462c-b4ce-d67f408e998a
// main.go:172: Chunk-Total 3
// main.go:173: Chunk-Size 768000
// main.go:174: Chunk-Number 1
// main.go:175: File-Path restfulobjects-spec.pdf
// main.go:176: Chunk-Hash 1aa87aba497b8ead0abaafda6d1fc7fc2a095ac8
// main.go:177: File-Hash
// main.go:178: Fail true
// main.go:232: finish
//
// main.go:154: /upload?upload_uid=2539e718-1f39-462c-b4ce-d67f408e998a&upload_path=%2Ftmp%2F
// main.go:155: map[User-Agent:[Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/53.0.2785.143 Chrome/53.0.2785.143 Safari/537.36] Chunk-Size:[768000] Content-Range:[bytes 768000-1536000/1963408] Accept-Encoding:[gzip, deflate, br] Accept-Language:[en-US,en;q=0.8] File-Path:[restfulobjects-spec.pdf] Chunk-Hash:[88d832bc4504c3b14babd85c532714227865ea78] Content-Type:[application/pdf] Accept:[*/*] Cookie:[_gat=1; _ga=GA1.2.52450644.1477966975] Content-Length:[768000] Origin:[https://code.hyperkube.co] Chunk-Number:[2] Chunk-Total:[3] Referer:[https://code.hyperkube.co/]]
// main.go:135: Header Empty File-Hash
// main.go:169: Upload Path: //tmp/
// main.go:170: Upload Uid: 2539e718-1f39-462c-b4ce-d67f408e998a
// main.go:172: Chunk-Total 3
// main.go:173: Chunk-Size 768000
// main.go:174: Chunk-Number 2
// main.go:175: File-Path restfulobjects-spec.pdf
// main.go:176: Chunk-Hash 88d832bc4504c3b14babd85c532714227865ea78
// main.go:177: File-Hash
// main.go:178: Fail true
// main.go:232: finish
//
// main.go:154: /upload?upload_uid=2539e718-1f39-462c-b4ce-d67f408e998a&upload_path=%2Ftmp%2F
// main.go:155: map[File-Hash:[bd6eb5e31100e6f652a3bf7bebf9d17fed4aba22] Content-Type:[application/pdf] Origin:[https://code.hyperkube.co] Chunk-Number:[3] Referer:[https://code.hyperkube.co/] Chunk-Size:[427408] Chunk-Total:[3] Accept:[*/*] Accept-Encoding:[gzip, deflate, br] Cookie:[_gat=1; _ga=GA1.2.52450644.1477966975] Content-Length:[427408] File-Path:[restfulobjects-spec.pdf] Chunk-Hash:[439f764241afe5988e752b2163f4db67ae8aaa0b] Accept-Language:[en-US,en;q=0.8] User-Agent:[Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/53.0.2785.143 Chrome/53.0.2785.143 Safari/537.36] Content-Range:[bytes 1536000-1963408/1963408]]
// main.go:169: Upload Path: //tmp/
// main.go:170: Upload Uid: 2539e718-1f39-462c-b4ce-d67f408e998a
// main.go:172: Chunk-Total 3
// main.go:173: Chunk-Size 427408
// main.go:174: Chunk-Number 3
// main.go:175: File-Path restfulobjects-spec.pdf
// main.go:176: Chunk-Hash 439f764241afe5988e752b2163f4db67ae8aaa0b
// main.go:177: File-Hash bd6eb5e31100e6f652a3bf7bebf9d17fed4aba22
// main.go:178: Fail false
// main.go:232: finish
//
