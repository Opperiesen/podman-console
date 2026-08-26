package ui

import (
	"fmt"
	"strings"

	"github.com/Opperiesen/podman-console/internal/domain"
)

func ContainerCreateView(data ViewData) string {
	lines := []string{TitleStyle.Render("PODMAN CONSOLE / NOUVEAU CONTENEUR")}
	lines = append(lines,
		fmt.Sprintf("Cible        %s", TitleStyle.Render(data.Profile.DisplayName())),
		fmt.Sprintf("Image        %s", display(data.ContainerCreateImageReference)),
		fmt.Sprintf("Image ID     %s", display(data.ContainerCreateImageID)),
		"",
	)

	fields := data.ContainerCreateFields
	name, command := "", ""
	if len(fields) > 0 {
		name = fields[0]
	}
	if len(fields) > 1 {
		command = fields[1]
	}
	nameValue := name
	commandValue := command
	if data.ContainerCreateFocus == 0 && !data.ContainerCreateRunning && !data.ContainerCreateRefreshing {
		nameValue += "▌"
	}
	if data.ContainerCreateFocus == 1 && !data.ContainerCreateRunning && !data.ContainerCreateRefreshing {
		commandValue += "▌"
	}
	lines = append(lines,
		fmt.Sprintf("%s%-12s %s", fieldMarker(data.ContainerCreateFocus == 0), "Nom", nameValue),
		fmt.Sprintf("%s%-12s %s", fieldMarker(data.ContainerCreateFocus == 1), "Commande", commandValue),
		MutedStyle.Render("Commande optionnelle · arguments uniquement · aucun shell"),
	)

	if data.ContainerCreateRunning || data.ContainerCreateRefreshing {
		lines = append(lines, "", containerCreateStatusLabel(data.ContainerCreateStatus))
	}
	if data.ContainerCreateError != nil {
		lines = append(lines, ErrorStyle.Render(data.ContainerCreateError.Error()))
	}
	if data.ContainerCreateResult.ContainerID != "" {
		state := "créé"
		if data.ContainerCreateResult.Started {
			state = "créé et démarré"
		} else {
			state = "créé mais non démarré"
		}
		lines = append(lines, "", fmt.Sprintf("Résultat      %s · ID %s", state, data.ContainerCreateResult.ContainerID))
		for _, warning := range data.ContainerCreateResult.Warnings {
			if strings.TrimSpace(warning) != "" {
				lines = append(lines, WarningStyle.Render("Avertissement : "+warning))
			}
		}
	}

	if data.ContainerCreateRunning || data.ContainerCreateRefreshing {
		lines = append(lines, "", MutedStyle.Render("Échap annuler la requête · ? aide · q quitter"))
	} else {
		lines = append(lines, "", MutedStyle.Render("Tab champ suivant · Entrée confirmer · Échap retour"))
	}
	return PanelStyle.Render(strings.Join(lines, "\n"))
}

func ContainerCreateConfirmation(data ViewData) string {
	request := data.ContainerCreateRequest
	imageReference := request.ImageReference
	if imageReference == "" {
		imageReference = "image sans référence"
	}
	command := strings.Join(request.Command, " ")
	if command == "" {
		command = "commande par défaut de l’image (aucune surcharge)"
	}
	return DialogStyle.Render(strings.Join([]string{
		WarningStyle.Render("CONFIRMATION REQUISE"),
		"",
		fmt.Sprintf("Cible        %s", TitleStyle.Render(data.Profile.DisplayName())),
		"Action       créer puis démarrer (détaché)",
		fmt.Sprintf("Image        %s", TitleStyle.Render(imageReference)),
		fmt.Sprintf("Image ID     %s", request.ImageID),
		fmt.Sprintf("Nom          %s", TitleStyle.Render(request.Name)),
		fmt.Sprintf("Commande     %s", command),
		MutedStyle.Render("Arguments transmis tels quels · aucun shell · aucun pull implicite"),
		"",
		MutedStyle.Render("Entrée/y confirmer · Échap/n annuler"),
	}, "\n"))
}

func fieldMarker(focused bool) string {
	if focused {
		return "› "
	}
	return "  "
}

func containerCreateStatusLabel(status domain.ContainerCreateStatus) string {
	switch status {
	case domain.ContainerCreateCreating:
		return StatusStyle.Render("création puis démarrage en cours…")
	case domain.ContainerCreateStarting:
		return StatusStyle.Render("démarrage en cours…")
	case domain.ContainerCreateRefreshing:
		return StatusStyle.Render("actualisation des conteneurs et images…")
	case domain.ContainerCreateSucceeded:
		return StatusStyle.Render("création réussie")
	case domain.ContainerCreatePartial:
		return WarningStyle.Render("créé mais non démarré")
	case domain.ContainerCreateCancelled:
		return WarningStyle.Render("opération annulée")
	case domain.ContainerCreateFailed:
		return ErrorStyle.Render("création échouée")
	default:
		return MutedStyle.Render("configuration en cours")
	}
}
