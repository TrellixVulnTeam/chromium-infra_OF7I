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
  [paramter(mandatory)]
  [String]$DiskImage,

  [paramter(mandatory)]
  [String]$SourceFile,

  [paramter(mandatory)]
  [String]$ImageDestinationPath,

  [String]$ImageType,

  [switch]$Force
)

Import-Module -Name 'WindowsImageManagement' -ErrorAction Stop | Out-Null

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

Copy-FileToDiskImage @params
