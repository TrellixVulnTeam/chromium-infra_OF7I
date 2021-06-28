<#
  .SYNOPSIS
    Wrapper script for the Add-FileToDiskImage function from the
    WindowsImageManagement module.

  .PARAMETER DiskImage
    The disk image file or folder to add the files to.

  .PARAMETER SourceFile
    The full path to the file or folder to be copied into the image.

  .PARAMETER ImageDestinationPath
    The path you want the file to be copied too from the root of the image.

  .PARAMETER ImageType
    The type of image to work on, ether install or boot. The default value is
    'Install'.

  .PARAMETER Force
    Override any files found at the ImageDestinationPath

  .EXAMPLE
    Copy the file C:\unattend.xml to the root of the disk image win10.iso.
    Use the default values for the ImageType and Index.

    PS> Copy-FileToDiskImage.ps1 -DiskImage C:\win10.iso -SourceFile C:\unattend.xml -ImageDestinationPath unattend.xml
#>

[cmdletbinding()]
param (
  [String]$DiskImage,

  [String]$SourceFile,

  [String]$ImageDestinationPath,

  [String]$ImageType,

  [switch]$Force
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
  if ('DiskImage' -notin $PSBoundParameters.keys) {
    throw 'DiskImage is a required Parameter'
  }

  if ('SourceFile' -notin $PSBoundParameters.keys) {
    throw 'SourceFile is a required Parameter'
  }

  if ('ImageDestinationPath' -notin $PSBoundParameters.keys) {
    throw 'ImageDestinationPath is a required Parameter'
  }
  #endregion Check for params

  $windows_image_management = "$PSScriptRoot\..\Modules\WindowsImageManagement"
  Import-Module -Name $windows_image_management -ErrorAction Stop | Out-Null

  $params = @{
    'DiskImage' = $DiskImage
    'SourceFile' = $SourceFile
    'ImageDestinationPath' = $ImageDestinationPath
    'Force' = $Force
    'InformationAction' = $InformationPreference
    'Verbose' = $VerbosePreference
    'ErrorAction' = 'Stop'
  }

  if ('ImageType' -in $PSBoundParameters.keys) {
    $params.Add('ImageType', $ImageType)
  }

  $invoke_obj.Output = Copy-FileToDiskImage @params

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