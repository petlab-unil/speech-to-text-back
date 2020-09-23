package server

import (
	"net/http"
	"speech-to-text-back/src/server/account"
	"strings"
)

type RouteTree struct {
	path     string
	children map[string]*RouteTree
	cb       func(*Handler, http.ResponseWriter, *http.Request)
}

func (t *RouteTree) RegisterRoute(path string, cb func(*Handler, http.ResponseWriter, *http.Request)) {
	branch := t
	trim := strings.Trim(path, "/")
	split := strings.Split(trim, "/")
	for _, p := range split {
		if p == "" {
			continue
		}
		if subBranch, ok := branch.children[p]; ok {
			branch = subBranch
		} else {
			newBranch := new(RouteTree)
			newBranch.children = make(map[string]*RouteTree)
			newBranch.path = p
			branch.children[p] = newBranch
			branch = newBranch
		}
	}
	branch.cb = cb
}

func (t *RouteTree) findPath(path []string, index int) *RouteTree {
	if subBranch, ok := t.children[path[index]]; ok {
		if index == len(path)-1 {
			return subBranch
		} else {
			return subBranch.findPath(path, index+1)
		}
	}
	return nil
}

func (t *RouteTree) ExecuteQuery(h *Handler, w http.ResponseWriter, r *http.Request) {
	trim := strings.Trim(r.URL.Path, "/")
	split := strings.Split(trim, "/")

	if split[0] != "account" {
		sessId := r.Header.Get("Authorization")
		if len(sessId) == 0 {
			sessId = r.URL.Query().Get("Authorization")
		}
		ok, err := account.CheckSession(sessId, h.MongoSession)
		if !ok {
			if err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
			} else {
				http.Error(w, "Invalid key", http.StatusForbidden)
			}
			return
		}
	}

	if subBranch := t.findPath(split, 0); subBranch != nil {
		subBranch.cb(h, w, r)
	} else {
		http.Error(w, "404 not found.", http.StatusNotFound)
	}
}
