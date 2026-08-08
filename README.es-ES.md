

<div align="center">
  <h1>cctrace</h1>
  <p><strong>Perfilador en tiempo real para agentes de código con IA. Envuelve Claude Code o Codex y observa cada evento en una línea de tiempo en cascada local.</strong></p>
  <p><em>Un único binario en Go. JSONL en disco, interfaz web en localhost. Sin demonios, sin telemetría, sin nube.</em></p>

  <p>
    <a href="https://github.com/androidZzT/cctrace/stargazers"><img src="https://img.shields.io/github/stars/androidZzT/cctrace?style=flat-square" alt="Stars"></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/androidZzT/cctrace?style=flat-square" alt="License"></a>
    <img src="https://img.shields.io/badge/go-1.22%2B-00ADD8?style=flat-square&logo=go" alt="Go">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey?style=flat-square" alt="Platform">
    <img src="https://img.shields.io/badge/local--first-no%20telemetry-orange?style=flat-square" alt="Local-first">
  </p>

  <p>
    <a href="#-why-cctrace">¿Por qué?</a> &bull;
    <a href="#-features">Características</a> &bull;
    <a href="#%EF%B8%8F-screenshots">Capturas de pantalla</a> &bull;
    <a href="#-quick-start">Inicio rápido</a> &bull;
    <a href="#-cli-reference">CLI</a> &bull;
    <a href="#%EF%B8%8F-architecture">Arquitectura</a> &bull;
    <a href="README_CN.md">中文</a>
  </p>
</div>

---

<p align="center">
  <img src="docs/screenshots/demo.gif" alt="cctrace waterfall timeline demo" width="900">
</p>

---

## 🤔 ¿Por qué cctrace?

Cuando ejecutas Claude Code o Codex en una tarea compleja, ocurre mucho que no puedes ver:

- qué llamadas a herramientas se ejecutaron y cuánto tiempo tomó cada una
- el orden de los eventos de proceso frente a los eventos de transcripción frente a los registros de ccglass
- cuándo el modelo pensó, se detuvo, reintentó o ramificó subagentes
- dónde se fue realmente el tiempo total

cctrace envuelve el proceso del agente, extrae eventos de la actividad del proceso, los archivos de transcripción y las trazas de ccglass en un modelo compartido, los correlaciona y muestra toda la sesión en una línea de tiempo en cascada local.

Sin demonios. Sin telemetría. JSONL en disco, una pequeña interfaz web en localhost.

## ✨ Características

- **Envuelve cualquier agente de código con IA** — `cctrace claude -- <cmd>` o `cctrace codex -- <cmd>` registran la ejecución de principio a fin
- **Modelo de traza unificado** — los registros de proceso, transcripción y ccglass se convierten en un único esquema de Evento / Sesión
- **Correlación heurística** — vincula eventos relacionados entre fuentes y etiqueta cada vínculo con una puntuación de confianza
- **Interfaz en cascada local** — abre la línea de tiempo en tu navegador, desplázate, haz zoom e inspecciona cualquier evento
- **Reproducible** — cada sesión es JSONL en disco; reprodúceme más tarde con `cctrace view <session-id>`
- **Un único binario en Go** — sin Python, sin Node, sin Docker, sin servicio en segundo plano

## 🖼️ Capturas de pantalla

La línea de tiempo en cascada para una sesión grabada — grupos de turnos a la izquierda, cada evento (usuario, IA, llamada a herramienta, resultado de herramienta) en un eje temporal compartido, con una vista general de densidad en la parte superior.

![cctrace waterfall timeline](docs/screenshots/waterfall-overview.png)

Haz clic en cualquier operación de herramienta para inspeccionar sus argumentos, salida, duración, IDs de correlación y el JSON crudo del evento.

![cctrace event details](docs/screenshots/event-details.png)

## 🚀 Inicio rápido

```sh
# Instalar desde el código fuente
go install github.com/androidZzT/cctrace/cmd/cctrace@latest

# Envolver una sesión de Claude Code
cctrace claude -- claude

# Envolver una sesión de Codex
cctrace codex -- codex

# Reproducir una sesión grabada
cctrace view <session-id>
```

Al envolver un agente se imprime la URL de la interfaz local de inmediato: ábrela para ver la cascada en vivo:

![cctrace launching and printing the local UI URL](docs/screenshots/cli-start.png)

## 📖 Referencia de la CLI

| Comando | Qué hace |
|---|---|
| `cctrace claude -- <cmd>` | Envuelve una invocación de Claude Code; registra eventos de proceso + transcripción + ccglass |
| `cctrace codex -- <cmd>` | Envuelve una invocación de Codex; misma canalización de registro |
| `cctrace view <session-id>` | Abre una sesión guardada en la interfaz web local |

Ejecuta `cctrace --help` para obtener todas las opciones.

## 🏗️ Arquitectura

Tres fuentes fluyen hacia un modelo compartido, se correlacionan con una puntuación de confianza y se muestran en la línea de tiempo local:

![cctrace data flow: sources to collectors to unified model to correlate to UI](docs/screenshots/dataflow.png)

```
cmd/cctrace                 CLI entry
└── internal/
    ├── app                 wires CLI parsing, collection, persistence, server startup
    ├── trace               shared Event / Session model
    ├── store               JSONL persistence under a session directory
    ├── collectors          process / transcript / ccglass → trace events
    ├── correlate           heuristic event linking with confidence tagging
    └── server              local web UI + event JSON API
```

Las sesiones se escriben como JSONL para que puedas usar grep, diff o enviarlas a otras herramientas sin iniciar cctrace.

## 🛠️ Desarrollo

```sh
# Ejecutar todas las pruebas
go test ./...

# Ejecutar la CLI en modo desarrollo
go run ./cmd/cctrace claude -- claude

# Prueba rápida de traza
go run ./cmd/cctrace claude -- sh -c 'printf cctrace-smoke'

# Reproducir
go run ./cmd/cctrace view <session-id>
```

## 📝 Licencia

[MIT](LICENSE)

## 🙏 Agradecimientos

cctrace lee registros de traza producidos por [ccglass](https://github.com/jianshuo/ccglass) (por [@jianshuo](https://github.com/jianshuo), MIT) como una de sus fuentes de eventos, y la ergonomía de `cctrace <agent> -- <cmd>` está inspirada en el enfoque de envolver e inspeccionar de ccglass. Si quieres ver exactamente lo que tu agente envía al modelo, revisa ccglass: se complementa bien con cctrace.

---

<div align="center">
  <sub>Construido con Go. Primero lo local. Sin telemetría.</sub>
</div>
