package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	_ "github.com/kamilkabir9/LDrop/statik" // NOTE: Replace with the absolute import path
	"github.com/mdp/qrterminal"
	"github.com/rakyll/statik/fs"
	"github.com/skratchdot/open-golang/open"
)

const (
	SuccessStatus = "Ok"
	FailedStatus  = "Err"
)

var uploadFolder string

type onlySuffixFilter struct {
	set    bool
	suffix []string
}

var oSF onlySuffixFilter

var verbose func(string)

func (filter *onlySuffixFilter) Set(value string) error {
	filter.set = true
	filter.suffix = strings.Split(value, ",")
	return nil
}
func (filter *onlySuffixFilter) filterFile(fileName string) bool {
	if filter.set {
		for _, v := range filter.suffix {
			if strings.HasSuffix(fileName, v) {
				return false
			}
		}
		return true
	}
	return false
}

func (filter *onlySuffixFilter) String() string {
	result := "\n"
	for i, v := range filter.suffix {
		result += fmt.Sprintf("%v:%v\n", i, v)
	}
	return result
}

//------------------------------------
type ignorePrefixFilter struct {
	set     bool
	preffix []string
}

var iPF ignorePrefixFilter

func (filter *ignorePrefixFilter) Set(value string) error {
	filter.set = true

	filter.preffix = strings.Split(value, ",")
	return nil
}
func (filter *ignorePrefixFilter) filterFile(fileName string) bool {
	if filter.set {
		for _, v := range filter.preffix {
			if strings.HasPrefix(fileName, v) {
				return true
			}
		}
		return false
	}
	return false
}

func (filter *ignorePrefixFilter) String() string {
	result := "\n"
	for i, v := range filter.preffix {
		result += fmt.Sprintf("%v:%v\n", i, v)
	}
	return result
}

//--------------------------
type ignoreSuffixFilter struct {
	set    bool
	suffix []string
}

var iSF ignoreSuffixFilter

func (filter *ignoreSuffixFilter) Set(value string) error {
	filter.set = true
	filter.suffix = strings.Split(value, ",")
	return nil
}

func (filter *ignoreSuffixFilter) filterFile(fileName string) bool {
	if filter.set {
		for _, v := range filter.suffix {
			if strings.HasSuffix(fileName, v) {
				return true
			}
		}
		return false
	}
	return false
}

func (filter *ignoreSuffixFilter) String() string {
	result := "\n"
	for i, v := range filter.suffix {
		result += fmt.Sprintf("%v:%v\n", i, v)
	}
	return result
}

func filterFile(fileName string) bool {
	if iSF.filterFile(fileName) || iPF.filterFile(fileName) || (ignoreHiddenFilesFlag && strings.HasPrefix(fileName, ".")) || oSF.filterFile(fileName) {
		return true
	}
	return false
}

var statikFS http.FileSystem
var ignoreHiddenFoldersFlag bool
var ignoreHiddenFilesFlag bool
var verboseFlag bool
var err error
var secretFlag string
var secretFlagMD5 string
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
var defaultSecret string

func checkSecret(secretEncoded string) bool {
	if secretEncoded == secretFlagMD5 {
		return true
	}
	verbose(fmt.Sprintf("Secret Failed %v=!%v !!!!!!", secretEncoded, secretFlagMD5))
	return false
}

// generate random secret
func randSecret(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	defaultSecret = randSecret(5)
	log.SetFlags(log.Lshortfile)
	flag.StringVar(&secretFlag, "secret", defaultSecret, "Pass secret code.")
	flag.Var(&iSF, "ignoreSuffix", "Pass file SUFFIX to exclude Example:\".png,.mp4\"")
	flag.Var(&iPF, "ignorePreffix", "Pass file PREFFIX to exclude Example:\"PIC-,MOV-\"")
	flag.Var(&oSF, "onlySuffix", "Pass file SUFFIX to only to include")
	flag.BoolVar(&ignoreHiddenFoldersFlag, "ignoreHiddenFolders", false, "Pass True to ignore hidden Files")
	flag.BoolVar(&ignoreHiddenFilesFlag, "ignoreHiddenFiles", false, "Pass True to ignore hidden Folders")
	flag.BoolVar(&verboseFlag, "v", false, "Pass true to print verbose info")
	flag.Parse()
	if verboseFlag {
		verbose = func(s string) {
			fmt.Println(s)
		}
	} else {
		verbose = func(s string) {
		}
	}
	hasher := md5.New()
	hasher.Write([]byte(secretFlag))
	secretFlagMD5 = hex.EncodeToString(hasher.Sum(nil))
	args := flag.Args()
	if len(args) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			log.Panicln(err)
		}
		uploadFolder, err = filepath.Abs(wd)
		if err != nil {
			log.Panicln(err)
		}
	} else {
		uploadFolder, err = filepath.Abs(args[0])
		if err != nil {
			log.Panicln(err)
		}
	}
	statikFS, err = fs.New()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/viewFile/", viewFileHandler)
	http.Handle("/", http.FileServer(statikFS))
	http.HandleFunc("/upload", upLoadHandler)               //api
	http.HandleFunc("/getLastFile", getLastFileHandler)     //api
	http.HandleFunc("/getAllFiles", getAllFilesHandler)     //api
	http.HandleFunc("/getFile/", getFileHandler)            //api
	http.HandleFunc("/downLoadFile/", serveThisFileHandler) //api

	//Adapted from https://stackoverflow.com/questions/43424787/how-to-use-next-available-port-in-http-listenandserve
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	fmt.Printf("Strarting Server...\n########################\nFolder:%v\nSecret:%v\nIP address: %v:%v\nverbose:%v\nFiltering rules\n------------------------\nIgnoring Files with Suffix(-ignoreSuffix):%v\nIgnoring Files with Preffix(-ignorePreffix):%v\nShowing Files ony with Suffix(-onlySuffix):%v\nHide Hidden Files(-ignoreHiddenFiles):%v\nHide Hidden Folders(-ignoreHiddenFolders):%v\n------------------------\n########################\n", uploadFolder, secretFlag, GetOutboundIP(), port, verboseFlag, iSF.String(), iPF.String(), oSF.String(), ignoreHiddenFilesFlag, ignoreHiddenFoldersFlag)
	fmt.Println("Scan QRCode to get IP address")
	qrterminal.GenerateHalfBlock(fmt.Sprintf("http://%v:%v", GetOutboundIP(), port), qrterminal.L, os.Stdout)
	open.Start(fmt.Sprintf("http://%v:%v", GetOutboundIP(), port))
	err = http.Serve(listener, nil)
	if err != nil {
		log.Println("ERR : ", err)
	}
}

//Adapted from https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	//log.Println("getting : " + r.URL.Path[1:])
	http.ServeFile(w, r, r.URL.Path[1:])
}

func UploadStatusJson(status string, desc string) string {
	type resultAsjson struct {
		Status      string
		Description string
	}
	var resultJson = resultAsjson{status, desc}
	result, err := json.Marshal(resultJson)
	if err != nil {
		log.Println("ERR : ", err)
		return fmt.Sprintf("{\"Status\":%v,\"Description\":%v}", FailedStatus, err)
	}
	return string(result)
}

func upLoadHandler(w http.ResponseWriter, r *http.Request) {

	verbose(fmt.Sprintln("Downloading File....."))
	file, fileHeader, err := r.FormFile("fileUpload")
	if err != nil {
		log.Println(err)
		result := UploadStatusJson(FailedStatus, fmt.Sprint(err))
		fmt.Fprint(w, result)
		return
	}
	if !checkSecret(r.Header.Get("secret")) {
		result := UploadStatusJson(FailedStatus, fmt.Sprintf("Error Uploading file %v. Secret mismatch !!!", fileHeader.Filename))
		fmt.Fprint(w, result)
		return
	}

	if _, err := os.Stat(uploadFolder); os.IsNotExist(err) {
		os.Mkdir(uploadFolder, 0777)
	}

	if err != nil {
		log.Println(err)
		result := UploadStatusJson(FailedStatus, fmt.Sprint(err))
		fmt.Fprint(w, result)
		return
	}
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		result := UploadStatusJson(FailedStatus, fmt.Sprint(err))
		fmt.Fprint(w, result)
		return
	}

	uniqfileName := getUniqFileName(fileHeader.Filename)
	of, err := os.Create(uniqfileName)
	if err != nil {
		log.Println("ERR : ", err)
		result := UploadStatusJson(FailedStatus, fmt.Sprint(err))
		fmt.Fprint(w, result)
		return
	}
	of.Write(fileBytes)

	verbose(fmt.Sprintf("File: %v saved at location: %v\n", fileHeader.Filename, uniqfileName))
	result := UploadStatusJson(SuccessStatus, fmt.Sprintf("Uploaded file %v", fileHeader.Filename))
	fmt.Fprint(w, result)
	verbose(fmt.Sprintln("Downloaded File : " + fileHeader.Filename))
}

//func getUniqFileName check if file with same file name exists .if yes then creates a new file name
func getUniqFileName(filename string) string {
	uploadFileName := filename
	uploadFileName = filepath.Join(uploadFolder, uploadFileName)
	exists := true
	count := 0
	for exists {
		count += 1
		if _, err := os.Stat(uploadFileName); os.IsNotExist(err) {
			exists = false
		} else {
			//file.png -> file-1.png
			uploadFileName = strings.Replace(uploadFileName, path.Ext(uploadFileName), "-"+strconv.Itoa(count)+path.Ext(uploadFileName), 1)
			verbose(fmt.Sprint("made uniq!!!!!!!!!!!!"))
		}
	}
	return uploadFileName
}

func getLastFileHandler(w http.ResponseWriter, r *http.Request) {
	if !checkSecret(r.Header.Get("secret")) {
		verbose("Got wrong secret from client")
		result := UploadStatusJson(FailedStatus, fmt.Sprintf("Error getting last file. Secret mismatch !!!"))
		fmt.Fprint(w, result)
		return
	}
	fileList := getAllFiles()
	lastFile := fileList[0]
	for _, file := range fileList {
		if lastFile.Info.ModTime.Before(file.Info.ModTime) {
			lastFile = file
		}
	}
	verbose(fmt.Sprintln("Last file:", lastFile.Name))
	//Adapted from https://stackoverflow.com/questions/31638447/how-to-server-a-file-from-a-handler-in-golang
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Content-Disposition", "attachment; filename="+lastFile.Info.Name)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, path.Join(uploadFolder, lastFile.Name))
}

type osFileInfo struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
}
type fileInfo struct {
	Name    string
	ModTime string
	Size    string
	Info    osFileInfo
}

var wg sync.WaitGroup
var mx sync.Mutex

func getAllFiles() []fileInfo {
	var fileNamesWithTime = new([]fileInfo)
	wg.Add(1)
	go getAllFilesConcurrent(uploadFolder, fileNamesWithTime)
	wg.Wait()
	verbose(fmt.Sprint("completed reading root Folder"))
	verbose(fmt.Sprint("Total nuber of Files: ", len(*fileNamesWithTime)))
	return *fileNamesWithTime
}
func getAllFilesConcurrent(Dir string, fileNamesWithTime *[]fileInfo) {
	verbose(fmt.Sprintln("Reading Dir: ", Dir))
	fileList, err := ioutil.ReadDir(Dir)
	if err != nil {
		log.Panicln("ERR : ", err)
	}
	for _, file := range fileList {

		if !file.IsDir() {
			var fileNameKey string
			if filterFile(file.Name()) {
				continue
			}
			fileNameKey = filepath.Join(Dir, file.Name())
			fileNameKey = strings.Replace(fileNameKey, uploadFolder+string(os.PathSeparator), "", 1)
			mx.Lock()
			*fileNamesWithTime = append(*fileNamesWithTime, fileInfo{fileNameKey, file.ModTime().Format(time.ANSIC), humanize.Bytes(uint64(file.Size())), osFileInfo{file.Name(), file.Size(), file.Mode(), file.ModTime(), file.IsDir()}})
			mx.Unlock()
		} else {
			if !(ignoreHiddenFoldersFlag && strings.HasPrefix(file.Name(), ".")) {
				wg.Add(1)
				go getAllFilesConcurrent(filepath.Join(Dir, file.Name()), fileNamesWithTime)
			} else if !ignoreHiddenFoldersFlag {
				wg.Add(1)
				go getAllFilesConcurrent(filepath.Join(Dir, file.Name()), fileNamesWithTime)
			}
		}
	}
	wg.Done()
}

func getAllFilesHandler(w http.ResponseWriter, r *http.Request) {
	verbose(fmt.Sprint("getting All files.."))
	if !checkSecret(r.Header.Get("secret")) {
		verbose("Got wrong secret from client")
		result := UploadStatusJson(FailedStatus, "secret mismatch")
		fmt.Fprintln(w, result)
		return
	}
	var fileNamesWithTime = getAllFiles()
	fileNamesJson, err := json.Marshal(fileNamesWithTime)
	if err != nil {
		log.Println("ERR : ", err)
		result := UploadStatusJson(FailedStatus, fmt.Sprint(err))
		fmt.Fprintln(w, result)
		return
	}
	result := UploadStatusJson(SuccessStatus, string(fileNamesJson))
	fmt.Fprintln(w, result)
	return
}

func getFileHandler(w http.ResponseWriter, r *http.Request) {
	u, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		panic(err)
	}
	fileName := u["fileName"][0]
	if fileName == "" {
		verbose("Error getting file name !!!")
		fmt.Fprint(w, fmt.Sprintf("Error getting file name from URL !!!"))
		return
	}
	fileName, err = url.QueryUnescape(fileName)
	if err != nil {
		log.Println(err)
	}
	secretRcvd := u["secret"][0]
	if secretRcvd == "" {
		verbose("Error getting file name !!!")
		fmt.Fprint(w, fmt.Sprintf("Error getting secret URL !!!"))
		return
	}
	if !checkSecret(secretRcvd) {
		verbose("Got wrong secret from client")
		fmt.Fprint(w, fmt.Sprintf("Error getting file:%v. Secret mismatch !!!", fileName))
		return
	}
	verbose(fmt.Sprintln("getting File : ", fileName))
	http.ServeFile(w, r, path.Join(uploadFolder, fileName))
	//http.ServeContent(w, r, path.Join(uploadFolder, fileName))

}
func serveThisFileHandler(w http.ResponseWriter, r *http.Request) {
	u, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		panic(err)
	}
	fileName := u["fileName"][0]
	if fileName == "" {
		verbose("Error getting file name !!!")
		fmt.Fprint(w, fmt.Sprintf("Error getting file name from URL !!!"))
		return
	}
	fileName, err = url.QueryUnescape(fileName)
	if err != nil {
		log.Println(err)
	}
	secretRcvd := u["secret"][0]
	if secretRcvd == "" {
		verbose("Error getting file name !!!")
		fmt.Fprint(w, fmt.Sprintf("Error getting secret URL !!!"))
		return
	}
	if !checkSecret(secretRcvd) {
		verbose("Got wrong secret from client")
		result := UploadStatusJson(FailedStatus, fmt.Sprintf("Error getting file:%v. Secret mismatch !!!", fileName))
		fmt.Fprint(w, result)
		return
	}
	verbose(fmt.Sprintln("serving File : ", fileName))
	//Adapted from https://stackoverflow.com/questions/31638447/how-to-server-a-file-from-a-handler-in-golang
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	//Adapted from https://stackoverflow.com/questions/13578428/duplicate-headers-received-from-server/25844041#25844041
	w.Header().Set("Content-Disposition", "attachment; filename="+"\""+fileName+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, path.Join(uploadFolder, fileName))
}
func viewFileHandler(w http.ResponseWriter, r *http.Request) {
	f, err := statikFS.Open("/viewFile.html")
	if err != nil {
		log.Println(err)
	}
	http.ServeContent(w, r, "viewFile.html", time.Now(), f)
}
