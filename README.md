# 💬 Hi — Social Working Media

[![Go Version](https://img.shields.io/github/go-mod/go-version/Oridjinnn/Hi---Social-Working-media)](https://golang.org/)
[![Build Status](https://img.shields.io/github/actions/workflow/status/Oridjinnn/Hi---Social-Working-media/ci.yml?branch=main)](https://github.com/Oridjinnn/Hi---Social-Working-media/actions)
[![License](https://img.shields.io/github/license/Oridjinnn/Hi---Social-Working-media)](LICENSE)

## Overview

Hi is a terminal-first collaboration tool designed for modern software teams and knowledge workers. It combines signal-driven discovery, lightweight chat, and shared workspace events into a single CLI application with a polished TUI.

### Key Features

- **Terminal-first workflow:** Fast keyboard-driven UI built with Bubble Tea and Lip Gloss.
- **Local Rewind:** Review recent activity, including visited signals, chat sessions, and GroupHouse events.
- **Signal-centric collaboration:** Manage short, actionable requests and connect with contributors quickly.
- **Workspace sharing:** Support for collaborative GroupHouse sessions and shared context.
- **GitHub integration:** Authentication and API-backed workflows for issues, comments, and notifications.

### Getting Started

#### Prerequisites

- Go 1.20 or later
- Git

#### Build and Run

```bash
git clone https://github.com/Oridjinnn/Hi---Social-Working-media.git
cd Hi---Social-Working-media/hi
go build -o hi .
./hi auth
```

The `auth` command starts the GitHub authentication flow and configures Hi for backend access.

### Rewind Preview

- Press `h` in the Feed tab to review recently visited signals and chat sessions.
- Press `h` in the GroupHouse tab to view recent workspace events.

### Repository Structure

- `cmd/` — CLI entry points and command definitions.
- `tui/` — Terminal UI components and application flow.
- `github/` — GitHub API client and integration logic.
- `grouphouse/` — Collaborative workspace and protocol implementation.
- `models/` — Domain models for signals, users, and events.
- `history/` — Local persistence for the Rewind/history feature.
- `config/`, `supabase/`, `notify/`, `utils/` — Supporting infrastructure and utilities.

### Development

```bash
go test ./...
gofmt -w .
```

CI runs `gofmt`, `golangci-lint`, `go build`, and `go test` on push and pull requests to `main`.

### Release

Tag a release like `v0.1.0` and the release workflow will build binaries and publish artifacts automatically.

### Contributing

1. Fork the repository.
2. Create a new branch: `git checkout -b feature/your-feature`.
3. Commit your changes with clear messages.
4. Open a pull request.

### License

This project is released under an open source license. See `LICENSE` for details.

---

# Bahasa Indonesia

## Ikhtisar

Hi adalah aplikasi kolaborasi terminal-first yang dirancang untuk tim perangkat lunak dan pekerja pengetahuan modern. Hi menggabungkan sinyal aktivitas, chat ringan, dan event workspace bersama ke dalam satu aplikasi CLI.

### Fitur Utama

- **Alur kerja terminal:** UI cepat yang dikendalikan dengan keyboard menggunakan Bubble Tea dan Lip Gloss.
- **Local Rewind:** Meninjau kembali aktivitas terbaru, termasuk sinyal yang dikunjungi, sesi chat, dan event GroupHouse.
- **Kolaborasi berbasis sinyal:** Mengelola permintaan singkat yang dapat segera ditindaklanjuti.
- **Berbagi workspace:** Mendukung sesi GroupHouse bersama dengan konteks kerja yang dibagi.
- **Integrasi GitHub:** Autentikasi dan workflow berbasis API GitHub untuk isu, komentar, dan notifikasi.

## Mulai Cepat

### Prasyarat

- Go 1.20 atau lebih baru
- Git

### Build dan Jalankan

```bash
git clone https://github.com/Oridjinnn/Hi---Social-Working-media.git
cd Hi---Social-Working-media/hi
go build -o hi .
./hi auth
```

Perintah `auth` memulai alur autentikasi GitHub dan mengonfigurasi Hi untuk akses backend.

### Rewind

- Tekan `h` di tab Feed untuk melihat sinyal dan sesi chat yang baru saja dikunjungi.
- Tekan `h` di tab GroupHouse untuk melihat event workspace terbaru.

## Struktur Repositori

- `cmd/` — Entrypoint CLI dan definisi perintah.
- `tui/` — Komponen UI terminal dan alur aplikasi.
- `github/` — Klien API GitHub dan integrasi.
- `grouphouse/` — Implementasi workspace kolaboratif.
- `models/` — Model domain untuk sinyal, pengguna, dan event.
- `history/` — Persistensi lokal untuk fitur Rewind.
- `config/`, `supabase/`, `notify/`, `utils/` — Infrastruktur dan utilitas pendukung.

## Pengembangan

```bash
go test ./...
gofmt -w .
```

CI menjalankan `gofmt`, `golangci-lint`, `go build`, dan `go test` pada push dan PR ke `main`.

## Rilis

Beri tag seperti `v0.1.0` dan workflow rilis akan membuat binary dan mempublikasikan artifacts secara otomatis.

## Kontribusi

1. Fork repositori.
2. Buat branch baru: `git checkout -b feature/nama-fitur`.
3. Commit perubahan dengan pesan yang jelas.
4. Buka pull request.

## Lisensi

Proyek ini dirilis di bawah lisensi open source. Lihat `LICENSE` untuk detail.

