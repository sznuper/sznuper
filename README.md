<p align="center">
  <img src="https://raw.githubusercontent.com/sznuper/.github/main/assets/readme-banner.png" style="max-width: 100%; width: 600px;" alt="Sznuper logo">
</p>

# Sznuper

A lightweight server monitor that runs healthchecks and sends notifications. Discord, Slack, Telegram, Teams, and more.

No dashboard. No database. No UI. Just a YAML config, a binary, and alerts delivered to Telegram, Slack, email, or any service supported by [Shoutrrr](https://shoutrrr.nickfedor.com/v0.14.0/services/overview/).

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/sznuper/dist/main/install.sh | sh
```

This installs the binary, runs `sznuper init` to create a config, and sets up a systemd service (root only).

To pin a specific version:

```sh
curl -fsSL https://raw.githubusercontent.com/sznuper/dist/main/install.sh | VERSION=v0.11.0 sh
```
