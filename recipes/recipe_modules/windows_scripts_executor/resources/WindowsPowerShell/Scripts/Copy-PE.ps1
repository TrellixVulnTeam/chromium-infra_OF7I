<#
  .SYNOPSIS
    Wrapper script for the Copy-PE function from the WindowsImageManagement
    module.

  .PARAMETER WinPeArch
    The architecture of the WinPE media folder to create. Supported values
    are: amd64, arm, arm64, x86.

  .PARAMETER Destination
    The full path and name of the WinPE media folder you are creating.

  .EXAMPLE
    Create amd64 winpe folder on your desktop

    PS> Copy-PE.ps1 -WinPeArch 'amd64' -Destination 'C:\Users\you\Desktop\new_winpe_amd64'
#>

[cmdletbinding()]
param (
  [Parameter(Mandatory)]
  [String]$WinPeArch,

  [Parameter(Mandatory)]
  [String]$Destination
)

$windows_image_management = "$PSScriptRoot\..\Modules\WindowsImageManagement"

Import-Module -Name $windows_image_management -ErrorAction Stop | Out-Null

$params = @{
  'WinPeArch' = $WinPeArch
  'Destination' = $Destination
  'InformationAction' = $InformationPreference
  'Verbose' = $VerbosePreference
  'ErrorAction' = 'Stop'
}

Copy-PE @params
