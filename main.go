package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"gitlab.com/milad.arab2010/dubai-backend/internal/apiserver"
	"gitlab.com/milad.arab2010/dubai-backend/internal/config"
	"gitlab.com/milad.arab2010/dubai-backend/internal/gpt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	cnf := config.LoadConfigOrPanic()

	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger().Level(zerolog.DebugLevel)

	sigs := make(chan os.Signal, 1)
	defer close(sigs)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	gptClient, err := gpt.NewClient(cnf.ChatGPT)
	if err != nil {
		panic(err)
	}

	server := apiserver.NewHttpServer(apiserver.ServerOptions{
		Port: cnf.Port,
	})

	server.HandleFunc("/prompt", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		bbyte, _ := ioutil.ReadAll(r.Body)
		chunks, err := gptClient.Prompt(string(bbyte))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		sb := strings.Builder{}
		for ch := range chunks {
			if ch.Err == io.EOF {
				break
			}

			if ch.Err != nil {
				log.Error().Err(ch.Err).Msg("data chunck is errornous")
				sb.WriteString(ch.Err.Error())
				break
			}
			sb.WriteString(ch.Content)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sb.String()))

		/* streaming
		for ch := range chunks {
			if ch.Err == io.EOF {
				return
			}

			if ch.Err != nil {
				log.Error().Err(ch.Err).Msg("data chunck is errornous")
				return
			}

			if _, err := w.Write([]byte(ch.Content)); err != nil {
				log.Error().Err(err).Msg("failed to write the response")
			}

			w.(http.Flusher).Flush()
		}
		*/

	})
	server.Start(sigs)
}
