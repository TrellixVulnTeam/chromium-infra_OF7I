function Close_DiskImageEnvironment {
  <#
    .SYNOPSIS
    Private function for performing the actions neccessary to close and save
    the disk image artifacts and environment.
  #>

  [cmdletbinding()]
  param (
    [parameter(Mandatory)]
    [DiskImageInfo]$DiskImageInfoObj,

    [Switch]$Force
  )

  if ($DiskImageInfoObj.WindowsImage) {
    Write-Information "Attempting to close windows image:`n$($DiskImageInfoObj.WindowsImage)"
    $close_WindowsImageEnvironment = @{
      'WindowsImageInfoObj' = $DiskImageInfoObj.WindowsImage
      'InformationAction' = $InformationPreference
      'Verbose' = $VerbosePreference
    }
    $DiskImageInfoObj.WindowsImage = Close_WindowsImageEnvironment @close_WindowsImageEnvironment
  }

  if ($DiskImageInfoObj.File) {
    Write-Information "Attempting to save disk image:`n$DiskImageInfoObj"
    $save_DiskImage = @{
      'DiskImageInfoObj' = $DiskImageInfoObj
      'Force' = $Force
      'InformationAction' = $InformationPreference
      'Verbose' = $VerbosePreference
    }
    $DiskImageInfoObj = Save-DiskImage @save_DiskImage
  }

  $DiskImageInfoObj | Write-Output
}
