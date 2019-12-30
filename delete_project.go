package randomize

import (
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
)

// DeleteProjectStep1 gets the project name from the user.
func DeleteProjectStep1(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		Serve404(w)
		return
	}

	ctx := r.Context()
	useremail := userEmail(r)

	projlist, err := getProjects(ctx, useremail, false)
	if err != nil {
		msg := "A database error occurred, your projects cannot be retrieved."
		log.Printf("Delete_project_step1: %v", err)
		rmsg := "Return to dashboard"
		messagePage(w, r, msg, rmsg, "/dashboard")
		return
	}

	if len(projlist) == 0 {
		msg := "You are not the owner of any projects.  A project can only be deleted by its owner."
		rmsg := "Return to dashboard"
		messagePage(w, r, msg, rmsg, "/dashboard")
		return
	}

	tvals := struct {
		User     string
		LoggedIn bool
		Proj     []*ProjectView
	}{
		User:     useremail,
		Proj:     formatProjects(projlist),
		LoggedIn: useremail != "",
	}

	if err := tmpl.ExecuteTemplate(w, "delete_project_step1.html", tvals); err != nil {
		log.Printf("deleteProjectStep1: %v", err)
	}
}

// DeleteProjectStep2 confirms that a project should be deleted.
func DeleteProjectStep2(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		Serve404(w)
		return
	}

	ctx := r.Context()
	useremail := userEmail(r)

	if err := r.ParseForm(); err != nil {
		ServeError(ctx, w, err)
		return
	}

	pkey := r.FormValue("project_list")
	svec := strings.Split(pkey, "::")

	tvals := struct {
		User        string
		LoggedIn    bool
		ProjectName string
		Pkey        string
		Nokey       bool
	}{
		User:     useremail,
		LoggedIn: useremail != "",
		Pkey:     pkey,
		Nokey:    len(pkey) == 0,
	}

	if len(svec) >= 2 {
		tvals.ProjectName = svec[1]
	}

	if err := tmpl.ExecuteTemplate(w, "delete_project_step2.html", tvals); err != nil {
		log.Printf("deleteProjectStep2: %v", err)
	}
}

// DeleteProjectStep3 deletes a project.
func DeleteProjectStep3(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		Serve404(w)
		return
	}

	ctx := r.Context()
	useremail := userEmail(r)
	pkey := r.FormValue("Pkey")

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		panic(err)
	}

	if !checkAccess(pkey, r) {
		msg := "You do not have access to this project."
		rmsg := "Return"
		messagePage(w, r, msg, rmsg, "/")
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("deleteProjectStep3 [1]: %v", err)
		ServeError(ctx, w, err)
		return
	}

	// Delete the SharingByProject object, but first read the
	// users list from it so we can delete the project from their
	// SharingByUsers records.
	doc, err := client.Doc("SharingByProject/" + pkey).Get(ctx)
	if err != nil {
		log.Printf("deleteProjectStep3 [3] %v", err)
		return
	}
	var sbp map[string]bool
	if err := doc.DataTo(&sbp); err != nil {
		log.Printf("deleteProjectStep3 [3] %v", err)
		return
	}

	// Delete the project
	if _, err := client.Doc("Project/" + pkey).Delete(ctx); err != nil {
		log.Printf("deleteProjectStep3 [3] %v", err)
		return
	}

	// Delete the sharing information
	if _, err := client.Doc("SharingByProject/" + pkey).Delete(ctx); err != nil {
		log.Printf("deleteProjectStep3 [3] %v", err)
		return
	}

	// Delete the project from each user's SharingByUser record.
	for user := range sbp {

		user = strings.ToLower(user)

		sbu := make(map[string]string)
		doc, err := client.Doc("SharingByUser/" + user).Get(ctx)
		if err != nil {
			log.Printf("Inconsistency in deleteProjectStep3: %v", err)
			return
		}
		if !doc.Exists() {
			log.Printf("Inconsistency in deleteProjectStep3: %v", err)
			return
		}
		if err := doc.DataTo(&sbu); err != nil {
			log.Printf("Inconsistency in deleteProjectStep3: %v", err)
			return
		}

		delete(sbu, pkey)

		if _, err = client.Doc("SharingByUser/"+user).Set(ctx, sbu); err != nil {
			log.Printf("deleteProjectStep3 [5]: %v", err)
			return
		}
	}

	tvals := struct {
		User     string
		LoggedIn bool
		Success  bool
	}{
		User:     useremail,
		LoggedIn: err == nil,
		Success:  useremail != "",
	}

	if err := tmpl.ExecuteTemplate(w, "delete_project_step3.html", tvals); err != nil {
		log.Printf("deleteProjectStep3 [9]: %v", err)
	}
}
