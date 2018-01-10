package main

import (
	"fmt"
	"net/http"
	"html/template"
	"github.com/eknkc/amber"
	"github.com/kelseyhightower/envconfig"
	"github.com/satori/go.uuid"
	"github.com/gorilla/mux"
	"playerElo"
)

type PostgresConf struct {
	Username string
	Password string
	Host string
	Database string
}

type TemplateConf struct {
	TemplateDirectory string
}

type Conf struct {
	PostgresConf
	TemplateConf
}

type WebHandler struct {
	elo *playerElo.PlayerRanking
	templates map[string]*template.Template
}

func NewWebHandler(conf *Conf) (*WebHandler, error) {
	dbstr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", conf.Username, conf.Password, conf.Host, conf.Database)
	elo, err := playerElo.NewPlayerRankings(dbstr, 1200, 24, 10)
	if err != nil {
		return nil, err
	}
	templateDir := conf.TemplateDirectory
	if templateDir == "" {
		templateDir = "templates/"
	}
	templates, err := amber.CompileDir(templateDir, amber.DefaultDirOptions, amber.DefaultOptions)
	if err != nil {
		return nil, err
	}
	toret := &WebHandler{
		elo : elo,
		templates : templates,
	}
	return toret, nil
}

func (wh *WebHandler) NewPlayer(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	game_name := r.Form.Get("game_name")
	realm_name := r.Form.Get("realm_name")
	kit_name := r.Form.Get("kit_name")
	err := wh.elo.NewPlayer(game_name + ", " + realm_name, kit_name)
	if err != nil {
		fmt.Fprintln(w, err)
	} else {
		fmt.Fprintln(w, "SUCCESS")
	}
}

func (wh *WebHandler) NewPlayerPage(w http.ResponseWriter, r *http.Request) {
	wh.templates["newplayer"].Execute(w, nil)
}

func (wh *WebHandler) NewMatchPage(w http.ResponseWriter, r *http.Request) {
	kit_id := mux.Vars(r)["kit_id"]
	results, err := wh.elo.FetchByKit(kit_id)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	data := struct{ Players []playerElo.Player; KitName string }{ Players : results, KitName : kit_id }
	wh.templates["newmatch"].Execute(w, &data)
}

func (wh *WebHandler) NewMatch(w http.ResponseWriter, r *http.Request) {
	kit_id := mux.Vars(r)["kit_id"]
	r.ParseForm()
	winner_val := r.Form.Get("winner")
	winner, err := uuid.FromString(winner_val)
	if err != nil {
		fmt.Fprintln(w, "Winner does not have a valid UUID.")
		return
	}
	loser_val := r.Form.Get("loser")
	loser, err := uuid.FromString(loser_val)
	if err != nil {
		fmt.Fprintln(w, "Loser does not have a valid UUID.")
		return
	}
	draw_val := r.Form.Get("draw")
	draw := draw_val == "draw"
	if winner != loser {
		err := wh.elo.RecordMatch(winner, loser, draw)
		if err != nil {
			fmt.Fprintln(w, err)
		}
	} else {
		fmt.Fprintln(w, "Same player entered in both fields")
		return
	}
	http.Redirect(w, r, "/rank/" + kit_id, http.StatusSeeOther)
}

func (wh *WebHandler) ViewPlayer(w http.ResponseWriter, r *http.Request) {
	player_id := mux.Vars(r)["player_id"]
	fmt.Fprintf(w, "VIEW PLAYER %s", player_id)
}

func (wh *WebHandler) ViewKit(w http.ResponseWriter, r *http.Request) {
	kit_id := mux.Vars(r)["kit_id"]
	results, err := wh.elo.FetchByKit(kit_id)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	data := struct{ Players []playerElo.Player; KitName string }{ Players : results, KitName : kit_id }
	wh.templates["viewkit"].Execute(w, &data)
}

func (wh *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	wh.templates["index"].Execute(w, nil)
}

func (wh *WebHandler) EloIndex(w http.ResponseWriter, r *http.Request) {
	wh.templates["elo_index"].Execute(w, nil)
}

func main() {
	var conf Conf
	envconfig.Process("bdr", &conf)
	wh, err := NewWebHandler(&conf)
	if err != nil {
		panic(err)
	}
	router := mux.NewRouter()
	router.HandleFunc("/", wh.Index).Methods("GET")
	router.HandleFunc("/elo", wh.EloIndex).Methods("GET")
	router.HandleFunc("/elo/addplayer", wh.NewPlayer).Methods("POST")
	router.HandleFunc("/elo/addplayer", wh.NewPlayerPage).Methods("GET")
	router.HandleFunc("/elo/addmatch/{kit_id}", wh.NewMatch).Methods("POST")
	router.HandleFunc("/elo/addmatch/{kit_id}", wh.NewMatchPage).Methods("GET")
	router.HandleFunc("/elo/player/{player_id}", wh.ViewPlayer).Methods("GET")
	router.HandleFunc("/elo/rank/{kit_id}", wh.ViewKit).Methods("GET")
	http.Handle("/", router)
	http.ListenAndServe(":8181", nil)
}
