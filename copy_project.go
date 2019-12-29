package randomize

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CopyProject(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		Serve404(w)
		return
	}

	ctx := r.Context()
	useremail := userEmail(r)
	pkey := r.FormValue("pkey")

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		panic(err)
	}

	ok := checkAccess(pkey, r)

	if !ok {
		msg := "Only the project owner can copy a project."
		rmsg := "Return to project dashboard"
		messagePage(w, r, msg, rmsg, "/project_dashboard?pkey="+pkey)
		return
	}

	doc, err := client.Doc("Project/" + pkey).Get(ctx)
	if err != nil {
		log.Printf("Copy_project: %v", err)
		msg := "Unknown database error."
		rmsg := "Return to project dashboard"
		messagePage(w, r, msg, rmsg, "/project_dashboard?pkey="+pkey)
		return
	}

	var proj Project
	if err := doc.DataTo(&proj); err != nil {
		log.Printf("Copy_project: %v", err)
		msg := "Unknown database error."
		rmsg := "Return to project dashboard"
		messagePage(w, r, msg, rmsg, "/project_dashboard?pkey="+pkey)
		return
	}

	tvals := struct {
		User        string
		LoggedIn    bool
		Pkey        string
		ProjectName string
	}{
		User:        useremail,
		LoggedIn:    useremail != "",
		Pkey:        pkey,
		ProjectName: proj.Name,
	}

	if err := tmpl.ExecuteTemplate(w, "copy_project.html", tvals); err != nil {
		log.Printf("copyProject failed to execute template: %v", err)
	}
}

func CopyProjectCompleted(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		Serve404(w)
		return
	}

	ctx := r.Context()
	useremail := userEmail(r)
	pkey := r.FormValue("pkey")

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		panic(err)
	}

	ok := checkAccess(pkey, r)

	if !ok {
		msg := "You do not have access to the requested project."
		rmsg := "Return to project dashboard"
		messagePage(w, r, msg, rmsg, "/project_dashboard?pkey="+pkey)
		return
	}

	doc, err := client.Doc("Project/" + pkey).Get(ctx)
	if err != nil {
		msg := "Unknown database error, the project was not copied."
		rmsg := "Return to dashboard"
		messagePage(w, r, msg, rmsg, "/dashboard")
		log.Printf("Copy_project: %v", err)
		return
	}

	var proj Project
	if err := doc.DataTo(&proj); err != nil {
		msg := "Unknown database error, the project was not copied."
		rmsg := "Return to dashboard"
		messagePage(w, r, msg, rmsg, "/dashboard")
		log.Printf("Copy_project: %v", err)
		return
	}

	// Check if the name is valid (not blank)
	newName := r.FormValue("new_project_name")
	newName = strings.TrimSpace(newName)
	if len(newName) == 0 {
		msg := "A name for the new project must be provided."
		rmsg := "Return to project dashboard"
		messagePage(w, r, msg, rmsg, "/project_dashboard?pkey="+pkey)
		return
	}
	proj.Name = newName

	// The owner of the copied project is the current user
	proj.Owner = useremail

	// Check if the project name has already been used.
	newkey := useremail + "::" + newName
	doc, err = client.Doc("Project/" + newkey).Get(ctx)
	if status.Code(err) == codes.NotFound {
		// OK
	} else if err != nil {
		msg := "Database error"
		rmsg := "Return to dashboard"
		messagePage(w, r, msg, rmsg, "/dashboard")
		return
	} else {
		msg := fmt.Sprintf("A project named \"%s\" belonging to user %s already exists.", newName, useremail)
		rmsg := "Return to dashboard"
		messagePage(w, r, msg, rmsg, "/dashboard")
		return
	}

	if _, err := client.Doc("Project/"+newkey).Set(ctx, &proj); err != nil {
		log.Printf("Copy_project: %v", err)
		msg := "Database error, the project was not copied."
		rmsg := "Return to dashboard"
		messagePage(w, r, msg, rmsg, "/dashboard")
		return
	}

	log.Printf("Copied %s to %s", pkey, newkey)
	msg := "The project has been successfully copied."
	rmsg := "Return to dashboard"
	messagePage(w, r, msg, rmsg, "/dashboard")
}
