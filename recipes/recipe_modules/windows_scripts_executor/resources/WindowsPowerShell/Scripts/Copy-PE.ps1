<#
  .SYNOPSIS
    Wrapper script for the Copy-PE function from the WindowsImageManagement
    module.

  .DESCRIPTION
    Script for performing the Copy-PE function for the Windows Image builders.

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
  [String]$WinPeArch,

  [String]$Destination
)

$invoke_obj = @{
  'Success' = $true
  'Output' = ''
  'ErrorInfo' = @{
    'Message' = ''
    'Line' = ''
    'PositionMessage' = ''
  }
}

try {
  #region Check for params
  if ('WinPeArch' -notin $PSBoundParameters.keys) {
    throw 'WinPeArch is a required Parameter'
  }

  if ('Destination' -notin $PSBoundParameters.keys) {
    throw 'Destination is a required Parameter'
  }
  #endregion Check for params

  $windows_image_management = "$PSScriptRoot\..\Modules\WindowsImageManagement"
  Import-Module -Name $windows_image_management -ErrorAction Stop | Out-Null

  $params = @{
    'WinPeArch' = $WinPeArch
    'Destination' = $Destination
    'InformationAction' = $InformationPreference
    'Verbose' = $VerbosePreference
    'ErrorAction' = 'Stop'
  }

  $invoke_obj.Output = Copy-PE @params

  Write-Information 'Convert to JSON else throw terminating error'
  $json = $invoke_obj | ConvertTo-Json -Compress -Depth 100 -ErrorAction Stop

} catch {
  $invoke_obj.Success = $false
  $invoke_obj.ErrorInfo.Message = $_.Exception.Message
  $invoke_obj.ErrorInfo.Line = $_.Exception.CommandInvocation.Line
  $invoke_obj.ErrorInfo.PositionMessage = $_.Exception.CommandInvocation.PositionMessage
  $json = $invoke_obj | ConvertTo-Json -Compress -Depth 100

} finally {
  Write-Information 'Write json result to stdout'
  $json
}