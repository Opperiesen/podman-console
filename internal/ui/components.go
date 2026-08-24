package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/Opperiesen/podman-console/internal/domain"
)

func TargetHeader(profile domain.ConnectionProfile, connected bool) string {
	name := profile.DisplayName()
	state := WarningStyle.Render("● déconnecté")
	if connected {
		state = StatusStyle.Render("● connecté")
	}
	uri := profile.URI
	if uri == "" {
		uri = "aucune cible configurée"
	}
	return TargetStyle.Render(fmt.Sprintf("CIBLE  %s  %s  %s", TitleStyle.Render(name), MutedStyle.Render(uri), state))
}

func ContainerRow(container domain.ContainerSummary, selected bool) string {
	state := stateStyle(container.State).Render(container.State.String())
	name := container.Name
	if name == "" {
		name = "<sans nom>"
	}
	image := container.Image
	if image == "" {
		image = "<image inconnue>"
	}
	row := fmt.Sprintf("%-24s %-12s %-28s %s", truncate(name, 24), shortID(container.ID), truncate(image, 28), state)
	if selected {
		return SelectedStyle.Render("› " + row)
	}
	return "  " + row
}

func ContainerHeader() string {
	return HeaderStyle.Render("  NOM                      ID           IMAGE                        ÉTAT")
}

func EmptyState(filter string) string {
	if filter != "" {
		return MutedStyle.Render(fmt.Sprintf("Aucun conteneur ne correspond à %q.", filter))
	}
	return MutedStyle.Render("Aucun conteneur sur cette cible.")
}

func ProfileList(profiles []domain.ConnectionProfile, active string, selected int) string {
	if len(profiles) == 0 {
		return MutedStyle.Render("Aucun profil. Appuyez sur n pour en créer un.")
	}
	lines := make([]string, 0, len(profiles))
	for i, profile := range profiles {
		marker := "  "
		if strings.EqualFold(profile.Name, active) {
			marker = "● "
		}
		line := fmt.Sprintf("%s%-24s %s", marker, truncate(profile.DisplayName(), 24), MutedStyle.Render(profile.URI))
		if i == selected {
			line = SelectedStyle.Render("› " + strings.TrimSpace(line))
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func DetailView(details *domain.ContainerDetails) string {
	if details == nil {
		return MutedStyle.Render("Chargement des détails…")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", HeaderStyle.Render("IDENTITÉ"))
	fmt.Fprintf(&b, "ID         %s\n", details.ID)
	fmt.Fprintf(&b, "Nom        %s\n", display(details.Name))
	fmt.Fprintf(&b, "Image      %s\n", display(details.Image))
	fmt.Fprintf(&b, "État       %s\n", stateStyle(details.State).Render(details.State.String()))
	fmt.Fprintf(&b, "Commande   %s\n", display(strings.Join(details.Command, " ")))
	fmt.Fprintf(&b, "Répertoire %s\n", display(details.WorkingDir))

	b.WriteString("\n")
	b.WriteString(HeaderStyle.Render("PORTS"))
	b.WriteString("\n")
	if len(details.Ports) == 0 {
		b.WriteString(MutedStyle.Render("aucun port publié"))
	} else {
		for _, port := range details.Ports {
			fmt.Fprintf(&b, "%s:%d → %d/%s\n", display(port.HostIP), port.HostPort, port.ContainerPort, display(port.Protocol))
		}
	}

	b.WriteString("\n")
	b.WriteString(HeaderStyle.Render("MONTAGES"))
	b.WriteString("\n")
	if len(details.Mounts) == 0 {
		b.WriteString(MutedStyle.Render("aucun montage"))
	} else {
		for _, mount := range details.Mounts {
			mode := mount.Mode
			if mode == "" {
				if mount.ReadWrite {
					mode = "rw"
				} else {
					mode = "ro"
				}
			}
			fmt.Fprintf(&b, "%s → %s (%s)\n", display(mount.Source), display(mount.Destination), mode)
		}
	}

	b.WriteString("\n")
	b.WriteString(HeaderStyle.Render("RÉSEAUX"))
	b.WriteString("\n")
	if len(details.Networks) == 0 {
		b.WriteString(MutedStyle.Render("aucun réseau"))
	} else {
		for _, network := range details.Networks {
			fmt.Fprintf(&b, "%s  %s  %s\n", display(network.Name), display(network.IPAddress), display(network.MACAddress))
		}
	}
	return b.String()
}

func StatsView(stats *domain.ContainerStats, stopped bool) string {
	if stats == nil {
		if stopped {
			return WarningStyle.Render("Le flux de métriques est arrêté ; aucune mesure reçue.")
		}
		return MutedStyle.Render("En attente de la première mesure…")
	}
	observed := stats.ObservedAt.Format(time.RFC3339)
	return strings.Join([]string{
		HeaderStyle.Render("ACTIVITÉ RESSOURCES"),
		fmt.Sprintf("CPU       %.2f%%", stats.CPUPercent),
		fmt.Sprintf("Mémoire   %s / %s (%.2f%%)", bytes(stats.MemoryUsageBytes), bytes(stats.MemoryLimitBytes), stats.MemoryPercent),
		MutedStyle.Render("Observation " + observed),
		func() string {
			if stopped {
				return WarningStyle.Render("Flux arrêté — dernière mesure conservée")
			}
			return StatusStyle.Render("Flux actif")
		}(),
	}, "\n")
}

func Confirmation(host, action, target, targetID string) string {
	return DialogStyle.Render(strings.Join([]string{
		WarningStyle.Render("CONFIRMATION REQUISE"),
		"", fmt.Sprintf("Cible : %s", TitleStyle.Render(host)),
		fmt.Sprintf("Voulez-vous %s le conteneur", action), TitleStyle.Render(target) + " ?",
		MutedStyle.Render("ID : " + targetID),
		"", MutedStyle.Render("Entrée/y confirmer · Échap/n annuler"),
	}, "\n"))
}

func ProfileForm(fields []string, focus int, err error) string {
	labels := []string{"Nom", "URI Podman", "Fichier d'identité (optionnel)"}
	values := make([]string, len(labels))
	copy(values, fields)
	lines := []string{HeaderStyle.Render("NOUVEAU PROFIL / MODIFICATION")}
	for i, label := range labels {
		prefix := "  "
		if i == focus {
			prefix = "› "
		}
		lines = append(lines, prefix+fmt.Sprintf("%-30s %s", label, values[i]))
	}
	lines = append(lines, "", MutedStyle.Render("Tab champ suivant · Ctrl+S enregistrer · Échap annuler"))
	if err != nil {
		lines = append(lines, ErrorStyle.Render(err.Error()))
	}
	return PanelStyle.Render(strings.Join(lines, "\n"))
}

func StatusLine(status string, err error) string {
	if err != nil {
		return ErrorStyle.Render("Erreur : " + err.Error())
	}
	if status != "" {
		return StatusStyle.Render(status)
	}
	return ""
}

func stateStyle(state domain.ContainerState) lipgloss.Style {
	switch state {
	case domain.StateRunning:
		return lipgloss.NewStyle().Foreground(Green)
	case domain.StateStopped, domain.StateExited:
		return lipgloss.NewStyle().Foreground(Yellow)
	case domain.StatePaused:
		return lipgloss.NewStyle().Foreground(Blue)
	default:
		return MutedStyle
	}
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func truncate(value string, width int) string {
	if lipgloss.Width(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	return value[:width-1] + "…"
}

func display(value string) string {
	if strings.TrimSpace(value) == "" {
		return MutedStyle.Render("indisponible")
	}
	return value
}

func bytes(value uint64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	div, exp := float64(unit), 0
	for n := value / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/div, "KMGTPE"[exp])
}
