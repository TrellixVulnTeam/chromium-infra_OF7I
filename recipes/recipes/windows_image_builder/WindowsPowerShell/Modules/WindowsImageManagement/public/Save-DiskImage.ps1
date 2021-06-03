function Save-DiskImage {
  <#
    .SYNOPSIS
    Create a disk image from a folder of source files

    .DESCRIPTION
    Creates a disk image using oscdimg and a folder of source files.

    .PARAMETER Folder
    Folder used as the source files for the creation of the disk image

    .PARAMETER Destination
    The Location to write the new disk image too

    .PARAMETER KeepFolder
    Switch for telling the command to keep the source files after successfully
    creating the disk image

    .PARAMETER Force
    Switch for forcing the deletion of a file found at the location specified
    for the new DiskImage

    .PARAMETER OscdimgName
    The name of the oscdimg binary. Include .exe

    .PARAMETER BootData
    The string to use for specifying the bootdata to oscdimg

    .EXAMPLE
    Save the contents of a folder to a disk image.

    PS> Save-DiskImage -Folder C:\win10_1809 -DiskImageDestination C:\win10_1809.iso

    .INPUT
    [String[]]

    .OUTPUT
    [DiskImageInfo]
  #>

  [cmdletbinding()]
  param (
    [parameter(
      Mandatory,
      ParameterSetName='non-DiskImageInfoObj')]
    [ValidateScript({Test-Path $_ -PathType Container})]
    [System.IO.FileInfo]$Folder,

    [parameter(ParameterSetName='non-DiskImageInfoObj')]
    [System.IO.FileInfo]$Destination,

    [parameter(
      Mandatory,
      ParameterSetName='DiskImageInfoObj')]
    [DiskImageInfo]$DiskImageInfoObj,

    [Switch]$KeepFolder,

    [Switch]$Force,

    [String]$OscdimgName = 'Oscdimg.exe',

    [String]$BootData
  )

  Test_OscdimgInPath -Name $OscdimgName

  if ('non-DiskImageInfoObj' -eq $PSCmdlet.ParameterSetName) {
    if ('Destination' -notin $PSBoundParameters.Keys) {
      $folder_name = $Folder | Split-Path -Leaf
      $Destination = "$env:TEMP\$folder_name`_modified.iso"
    }

    $disk_image_info = [DiskImageInfo]::new($Destination, $Folder)
  }

  if ('DiskImageInfoObj' -eq $PSCmdlet.ParameterSetName) {
    $disk_image_info = $DiskImageInfoObj
  }

  if ('BootData' -notin $PSBoundParameters.Keys) {
    $BootData = "2#p0,e,b$($disk_image_info.ContentLocation)\boot\etfsboot.com#pEF,e,b$($disk_image_info.ContentLocation)\efi\microsoft\boot\efisys.bin"
  }

  if (-not ($disk_image_info.File)){
    throw "Disk Image File location is null or empty"
  }

  if (-not ($disk_image_info.ContentLocation)){
    throw "Disk Image Content Location is null or empty"
  }

  if (Test-Path $disk_image_info.File) {
    Write-Information "Pre-existing file found at $($disk_image_info.File.FullName)"
    if ($Force) {
      Write-Information "Attempting to remove file $($disk_image_info.File.FullName)"
      Remove-Item $disk_image_info.File -Recurse -Force -Confirm:$false
    }
    else {
      throw "Pre-existing file found at $($disk_image_info.File)"
    }
  }

  # TODO(actodd): Find a cleaner way to supress or redirect stdout that is
  # showing up as stderror. Its adding +100 lines of output.
  $starting_error_action_preference = $ErrorActionPreference
  $ErrorActionPreference = 'SilentlyContinue'
  try {
    $null = & $OscdimgName -m -o -u2 -udfver102 -bootdata:$bootdata $disk_image_info.ContentLocation $disk_image_info.File
  }
  catch {
    Write-Error $Error[0]
  }
  $ErrorActionPreference = $starting_error_action_preference

  if (-not(Test-Path $disk_image_info.File)) {
    throw "Failed to save disk image to $($disk_image_info.File.FullName)"
  }

  if ($KeepFolder) {
    Write-Information "Leaving disk image folder: $($disk_image_info.ContentLocation)"
  }
  else {
    Write-information "Deleting disk image folder: $($disk_image_info.ContentLocation.FullName)"
    Remove-Item -Path $disk_image_info.ContentLocation -Recurse -Force -Confirm:$false

    if (Test-Path $disk_image_info.ContentLocation) {
      Write-Warning "Failed to remove item $($disk_image_info.ContentLocation)"
    }
    else {
      $disk_image_info.ContentLocation = $null
    }
  }

  $disk_image_info | Write-Output
}