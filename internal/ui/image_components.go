package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Opperiesen/podman-console/internal/domain"
)

func ImageHeader() string {
	return HeaderStyle.Render("  RÉFÉRENCE                    ID           DIGEST             TAILLE      CRÉÉ             ÉTAT")
}

func ImageRow(image domain.ImageSummary, selected bool) string {
	reference := image.PrimaryReference()
	if reference == "" {
		reference = "<sans référence>"
	}
	digest := image.DisplayDigest()
	if digest == "" {
		digest = "—"
	} else {
		digest = shortDigest(digest)
	}
	created := "—"
	if !image.CreatedAt.IsZero() {
		created = image.CreatedAt.Local().Format("2006-01-02 15:04")
	}
	state := imageState(image)
	row := fmt.Sprintf("%-28s %-12s %-18s %-10s %-16s %s",
		truncate(reference, 28), shortID(image.ID), truncate(digest, 18), bytes(image.Size), created, state)
	if selected {
		return SelectedStyle.Render("› " + row)
	}
	return "  " + row
}

func ImageEmptyState(filter string) string {
	if strings.TrimSpace(filter) != "" {
		return MutedStyle.Render(fmt.Sprintf("Aucune image ne correspond à %q.", filter))
	}
	return MutedStyle.Render("Aucune image sur cette cible. Vous pouvez lancer un téléchargement avec P.")
}

func ImageDetailView(details *domain.ImageDetails, loading bool) string {
	if details == nil || loading {
		return MutedStyle.Render("Chargement des détails image…")
	}
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("IDENTITÉ"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "ID             %s\n", display(details.ID))
	fmt.Fprintf(&b, "Parent         %s\n", display(details.ParentID))
	fmt.Fprintf(&b, "Architecture   %s\n", display(details.Architecture))
	fmt.Fprintf(&b, "OS             %s\n", display(details.OS))
	fmt.Fprintf(&b, "Taille         %s\n", bytes(details.Size))
	created := "indisponible"
	if !details.CreatedAt.IsZero() {
		created = details.CreatedAt.Local().Format("2006-01-02 15:04:05 MST")
	}
	fmt.Fprintf(&b, "Créée          %s\n", created)
	fmt.Fprintf(&b, "Conteneurs     %d\n", details.Containers)

	b.WriteString("\n")
	b.WriteString(HeaderStyle.Render("RÉFÉRENCES"))
	b.WriteString("\n")
	if len(details.References) == 0 {
		b.WriteString(MutedStyle.Render("aucune référence — image probablement dangling"))
	} else {
		for _, reference := range details.References {
			fmt.Fprintf(&b, "%s\n", display(reference))
		}
	}

	b.WriteString("\n")
	b.WriteString(HeaderStyle.Render("DIGESTS"))
	b.WriteString("\n")
	if len(details.Digests) == 0 && details.Digest == "" {
		b.WriteString(MutedStyle.Render("aucun digest"))
	} else {
		for _, digest := range details.Digests {
			fmt.Fprintf(&b, "%s\n", display(digest))
		}
		if len(details.Digests) == 0 && details.Digest != "" {
			fmt.Fprintf(&b, "%s\n", details.Digest)
		}
	}

	b.WriteString("\n")
	b.WriteString(HeaderStyle.Render("LABELS"))
	b.WriteString("\n")
	if len(details.Labels) == 0 {
		b.WriteString(MutedStyle.Render("aucun label"))
	} else {
		keys := make([]string, 0, len(details.Labels))
		for key := range details.Labels {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&b, "%s=%s\n", key, details.Labels[key])
		}
	}
	return b.String()
}

func ImagePullView(data ViewData, height int) string {
	lines := []string{TitleStyle.Render("TÉLÉCHARGER UNE IMAGE")}
	if data.ImagePullInputEditing {
		lines = append(lines, HeaderStyle.Render("RÉFÉRENCE")+"  "+data.ImagePullInput+"▌")
	} else if data.ImagePullReference != "" {
		lines = append(lines, HeaderStyle.Render("RÉFÉRENCE")+"  "+data.ImagePullReference)
	} else {
		lines = append(lines, HeaderStyle.Render("RÉFÉRENCE")+"  "+MutedStyle.Render("saisissez une référence registry"))
	}

	state := imagePullStatusLabel(data.ImagePullStatus)
	if data.ImagePulling {
		state = StatusStyle.Render("flux actif")
	}
	lines = append(lines, state)

	if data.ImagePullError != nil {
		lines = append(lines, ErrorStyle.Render(data.ImagePullError.Error()))
	}
	if len(data.ImagePullEvents) == 0 {
		lines = append(lines, MutedStyle.Render("Aucune progression reçue."))
	} else {
		lines = append(lines, HeaderStyle.Render("PROGRESSION (ordre d’arrivée)"))
		progress := make([]string, 0, len(data.ImagePullEvents))
		for _, event := range data.ImagePullEvents {
			text := event.Text
			if text == "" && event.Kind == domain.ImagePullSuccess {
				text = "image disponible selon Podman"
			}
			if text == "" {
				continue
			}
			progress = append(progress, imagePullEventText(event.Kind, text))
		}
		lines = append(lines, strings.Join(progress, ""))
	}
	maxLines := height - 8
	if maxLines < 4 {
		maxLines = 4
	}
	if len(lines) > maxLines {
		lines = append(lines[:3], lines[len(lines)-(maxLines-3):]...)
	}
	return strings.Join(lines, "\n")
}

func ResourceConfirmation(host, action, resource, target, targetID string) string {
	article := "le " + resource
	if resource == "image" {
		article = "l’image"
	}
	return DialogStyle.Render(strings.Join([]string{
		WarningStyle.Render("CONFIRMATION REQUISE"),
		"", fmt.Sprintf("Cible : %s", TitleStyle.Render(host)),
		fmt.Sprintf("Voulez-vous %s %s", action, article), TitleStyle.Render(target) + " ?",
		MutedStyle.Render("ID : " + targetID),
		"", MutedStyle.Render("Entrée/y confirmer · Échap/n annuler"),
	}, "\n"))
}

func imageTarget(image domain.ImageSummary) string {
	if reference := image.PrimaryReference(); reference != "" {
		return reference
	}
	if image.ID != "" {
		return shortID(image.ID)
	}
	return "image"
}

func imageState(image domain.ImageSummary) string {
	if image.Dangling {
		return WarningStyle.Render("dangling")
	}
	if image.ReadOnly {
		return MutedStyle.Render("read-only")
	}
	return StatusStyle.Render("local")
}

func shortDigest(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "sha256:") && len(value) > len("sha256:")+12 {
		return value[:len("sha256:")+12]
	}
	return value
}

func imagePullStatusLabel(status domain.ImageOperationStatus) string {
	switch status {
	case domain.ImageOperationSucceeded:
		return StatusStyle.Render("téléchargement réussi")
	case domain.ImageOperationFailed:
		return ErrorStyle.Render("téléchargement échoué")
	case domain.ImageOperationCancelled:
		return WarningStyle.Render("flux arrêté — progression conservée")
	case domain.ImageOperationRefreshing:
		return StatusStyle.Render("actualisation de l’inventaire…")
	default:
		return MutedStyle.Render("en attente")
	}
}

func imagePullEventText(kind domain.ImagePullEventKind, text string) string {
	switch kind {
	case domain.ImagePullError:
		return ErrorStyle.Render("[erreur] ") + text
	case domain.ImagePullCancelled:
		return WarningStyle.Render("[annulé] ") + text
	default:
		return text
	}
}
