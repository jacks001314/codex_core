param(
  [string]$BaseUrl = "",
  [switch]$Live
)
$ErrorActionPreference = "Stop"
if ($Live) { $env:CODEX_LIVE_TEST = "1" }
if ($BaseUrl) { $env:OPENAI_BASE_URL = $BaseUrl }
go test ./integration -count=1 -v
