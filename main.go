package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
)

type ReceiverOptions struct {
	Host string
	Port int
}

func mkdirAll(path string) bool {
	dirpath := filepath.Dir(path)
	_, err := os.Stat(dirpath)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		if err = os.MkdirAll(dirpath, os.ModePerm); err != nil {
			return false
		}
		return true
	}
	return false
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("I'm ready for that, you know."))
		return
	}
	err := r.ParseMultipartForm(int64(10 << 20))
	if err != nil {
		glog.Error("parse form failed.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	saveto := r.Form.Get("to")
	if !filepath.IsAbs(saveto) || !mkdirAll(saveto) {
		glog.Errorf("create dest dir %s failed.", saveto)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	infile, _, err := r.FormFile("file")
	if err != nil {
		glog.Error("read form file failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	outfile, err := os.OpenFile(saveto, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		glog.Errorf("create file %s failed. error: %v", saveto, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer outfile.Close()
	_, err = io.Copy(outfile, infile)
	if err != nil {
		glog.Error("copy file failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	glog.V(4).Infof("%s saved.", saveto)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("1"))
}

func (o *ReceiverOptions) addFlags(fs *pflag.FlagSet) {
	fs.IntVarP(&o.Port, "port", "p", 8200, "监听端口")
	fs.StringVarP(&o.Host, "listen", "l", "0.0.0.0", "监听地址")
}

func startServer(o *ReceiverOptions) {
	router := mux.NewRouter()
	router.HandleFunc("/", handler)
	mux := http.NewServeMux()
	mux.Handle("/", router)
	http.ListenAndServe(fmt.Sprintf("%s:%d", o.Host, o.Port), mux)
}

func initFlags() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	flag.CommandLine.Parse([]string{})
	pflag.VisitAll(func(flag *pflag.Flag) {
		glog.V(4).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}

func main() {
	opts := &ReceiverOptions{}
	opts.addFlags(pflag.CommandLine)
	initFlags()
	startServer(opts)
}
