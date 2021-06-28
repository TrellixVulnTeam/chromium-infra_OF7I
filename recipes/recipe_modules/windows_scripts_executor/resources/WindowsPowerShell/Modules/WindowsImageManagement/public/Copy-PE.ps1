function Copy-PE {
  <#
    .SYNOPSIS
      PowerShell port of copype.cmd used for making WinPE media folders.

    .PARAMETER WinPeArch
      The architecture of the WinPE media folder to create. Supported values
      are: amd64, arm, arm64, x86.

    .PARAMETER Destination
      The full path and name of the WinPE media folder you are creating.

    .EXAMPLE
      Create amd64 winpe folder on your desktop

      PS> Copy-PE -WinPeArch 'amd64' -Destination 'C:\Users\you\Desktop\new_winpe_amd64'
  #>

  [cmdletbinding()]
  param (
    [parameter(Mandatory)]
    [ValidateSet('amd64', 'arm', 'arm64', 'x86')]
    [String]$WinPeArch,

    [parameter(Mandatory)]
    [System.IO.FileInfo]$Destination
  )

  $TEMPL = 'media'
  $FWFILES = 'fwfiles'
  $SOURCE = "$env:WinPERoot\$WinPeArch"
  $FWFILESROOT = "$env:OSCDImgRoot\..\..\$WinPeArch\Oscdimg"
  $WIMSOURCEPATH= "$SOURCE\en-us\winpe.wim"

  if (-not (Test-Path $env:WinPERoot)) {
    throw "Failed to find WinPERoot dir at $env:WinPERoot"
  }

  Write-Information 'Check for source material directories'
  if (-not (Test-Path $SOURCE)) {
    $msg = "Failed to find source dir: $SOURCE`n" +
      "WinPERoot: $env:WinPERoot`n" +
      "WinPEArch: $WinPEArch`n" +
      "WinPERoot contents:`n" +
      (Get-Childitem -Path $env:WinPERoot).fullname
    throw $message
  }

  if (-not (Test-Path $FWFILESROOT)) {
    throw "The following path for firmware files was not found: $FWFILESROOT"
  }

  if (-not (Test-Path $WIMSOURCEPATH)) {
    throw "WinPE WIM file does not exist: $WIMSOURCEPATH"
  }

  Write-Information 'Create Windows PE working directories'
  $directories = @(
    $Destination,
    (Join-Path $Destination $TEMPL),
    (Join-Path $Destination 'mount'),
    (Join-Path $Destination $FWFILES)
  )

  foreach ($dir in $directories) {
    if (-not (Test-Path $dir)) {
      Write-information "Directory not found. Creating directory: $dir"
      $null = New-Item -Path $dir -ItemType 'Directory' -Force
    }

    if (-not (Test-Path $dir)) {
      throw "Failed to create directory: $dir"
    }
  }

  Write-Information 'Copy the boot files and WinPE WIM to the destination location'
  $source_files = (Join-Path $SOURCE 'Media\*')
  $destination_location = (Join-Path $Destination $TEMPL)
  $copy_item_params = @{
    'Path' = $source_files
    'Destination' = $destination_location
    'Recurse' = $true
    'Force' = $true
    'Confirm' = $false
  }
  Copy-Item @copy_item_params

  $dir = "$Destination\$TEMPL\sources"
  if (-not (Test-Path $dir)) {
    Write-information "Directory not found. Creating directory: $dir"
    $null = New-Item -Path $dir -ItemType 'Directory' -Force
  }

  if (-not (Test-Path $dir)) {
    throw "Failed to create directory: $dir"
  }

  $destination_location = "$Destination\$TEMPL\sources\boot.wim"
  $copy_item_params = @{
    'Path' = $WIMSOURCEPATH
    'Destination' = $destination_location
    'Force' = $true
    'Confirm' = $false
  }
  Copy-Item @copy_item_params

  $dir = "$Destination\$TEMPL\sources\boot.wim"
  if (-not (Test-Path $dir)) {
    Write-information "Directory not found. Creating directory: $dir"
    $null = New-Item -Path $dir -ItemType 'Directory' -Force
  }

  if (-not (Test-Path $dir)) {
    throw "Failed to create directory: $dir"
  }

  Write-Information 'Copy the boot sector files to enable ISO creation and boot'

  $destination_location = (Join-Path $Destination $FWFILES)
  Write-Information "Boot sector files destination set to $destination_location"

  $efisys_file = Join-Path $FWFILESROOT "efisys.bin"
  Write-Information "Attempting to copy $efisys_file"

  $copy_item_params = @{
    'Path' = $efisys_file
    'Destination' = $destination_location
    'Force' = $true
    'Confirm' = $false
  }
  Copy-Item @copy_item_params

  $etfsboot_file = Join-Path $FWFILESROOT "etfsboot.com"
  if (Test-Path $etfsboot_file) {
    Write-Information "Found $etfsboot_file. Attempting to copy."
    $copy_item_params = @{
      'Path' = $etfsboot_file
      'Destination' = $destination_location
      'Force' = $true
      'Confirm' = $false
    }
    Copy-Item @copy_item_params
  }
}