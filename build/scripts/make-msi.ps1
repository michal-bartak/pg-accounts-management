<#
.SYNOPSIS
  Build the Windows MSI installer from the .exe in build/bin, using the WiX v3 Toolset.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File build/scripts/make-msi.ps1 -Version 0.3.0

  Run from the repo root. Compiles build/windows/installer/product.wxs (candle + light)
  after generating the two WiX UI bitmaps from build/appicon.png, and writes
  dist/<Output>-v<Version>-windows-<ArchLabel>.msi.
#>
[CmdletBinding()]
param(
  [string]$Output    = "DbAccounts",
  [string]$Version   = "",
  [string]$ArchLabel = "amd64",
  [string]$DistDir   = "dist"
)

$ErrorActionPreference = "Stop"

if (-not $Version) { $Version = (Get-Content VERSION -Raw).Trim() }

$exePath = Join-Path $PWD "build\bin\$Output.exe"
if (-not (Test-Path $exePath)) { throw "$exePath not found - run 'make build' first" }

$iconPath = Join-Path $PWD "build\windows\icon.ico"
if (-not (Test-Path $iconPath)) { throw "$iconPath not found" }

$wxs        = Join-Path $PWD "build\windows\installer\product.wxs"
$licenseRtf = Join-Path $PWD "build\windows\installer\License.rtf"

# MSI ProductVersion must be numeric x.y.z.b — derive it from the semver in VERSION.
$m = [regex]::Match($Version, '^(\d+)(?:\.(\d+))?(?:\.(\d+))?')
if (-not $m.Success) { throw "Cannot derive an MSI version from '$Version'" }
$part = { param($g) if ($g.Success -and $g.Value) { $g.Value } else { "0" } }
$wixVersion = "{0}.{1}.{2}.0" -f (& $part $m.Groups[1]), (& $part $m.Groups[2]), (& $part $m.Groups[3])

$wix = Get-ChildItem "C:\Program Files (x86)\WiX Toolset*\bin", "C:\Program Files\WiX Toolset*\bin" `
         -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty FullName
if (-not $wix) { throw "WiX Toolset bin directory not found - install it with: choco install wixtoolset" }

# ── UI bitmaps, drawn from the app icon so they track the icon automatically ──
Add-Type -AssemblyName System.Drawing
$src   = [System.Drawing.Image]::FromFile((Join-Path $PWD "build\appicon.png"))
$brand = [System.Drawing.Color]::FromArgb(255, 222, 233, 252)   # same tint as the macOS DMG
$gray  = [System.Drawing.Color]::FromArgb(255, 240, 240, 240)   # Windows default dialog bg
$sm    = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
$im    = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic

# Banner: 493x58 — the strip across the top of every installer page.
$bmp = New-Object System.Drawing.Bitmap(493, 58)
$g   = [System.Drawing.Graphics]::FromImage($bmp)
$g.SmoothingMode = $sm; $g.InterpolationMode = $im
$g.Clear($brand)
$g.DrawImage($src, 493 - 52, 5, 46, 46)
$g.Dispose()
$bannerBmp = Join-Path $env:TEMP "wix-banner.bmp"
$bmp.Save($bannerBmp, [System.Drawing.Imaging.ImageFormat]::Bmp); $bmp.Dispose()

# Dialog: 493x312 — Welcome/Finish screens. Left 164px brand panel, rest dialog gray.
$bmp = New-Object System.Drawing.Bitmap(493, 312)
$g   = [System.Drawing.Graphics]::FromImage($bmp)
$g.SmoothingMode = $sm; $g.InterpolationMode = $im
$g.Clear($gray)
$g.FillRectangle((New-Object System.Drawing.SolidBrush($brand)), 0, 0, 164, 312)
$g.DrawImage($src, (164 - 130) / 2, (312 - 130) / 2, 130, 130)
$g.Dispose()
$dialogBmp = Join-Path $env:TEMP "wix-dialog.bmp"
$bmp.Save($dialogBmp, [System.Drawing.Imaging.ImageFormat]::Bmp); $bmp.Dispose()
$src.Dispose()

# ── Compile ──────────────────────────────────────────────────────────────────
New-Item -ItemType Directory -Force -Path $DistDir | Out-Null
$msi     = Join-Path $DistDir "$Output-v$Version-windows-$ArchLabel.msi"
$wixobj  = Join-Path $env:TEMP "product.wixobj"
Remove-Item $msi -ErrorAction SilentlyContinue

& "$wix\candle.exe" -nologo -arch x64 `
  -dVersion="$wixVersion" `
  -dExePath="$exePath" `
  -dIconPath="$iconPath" `
  -dBannerBmp="$bannerBmp" `
  -dDialogBmp="$dialogBmp" `
  -dLicenseRtf="$licenseRtf" `
  -o $wixobj $wxs
if ($LASTEXITCODE -ne 0) { throw "candle.exe failed with exit code $LASTEXITCODE" }

& "$wix\light.exe" -nologo -spdb -ext WixUIExtension -o $msi $wixobj
if ($LASTEXITCODE -ne 0) { throw "light.exe failed with exit code $LASTEXITCODE" }

Write-Output $msi
