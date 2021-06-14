function Open_DiskImageEnvironment {
  <#
    .SYNOPSIS
    Private function for performing the actions neccessary to open the disk
    image artifacts and environment.
  #>

  [cmdletbinding()]
  param (
    [parameter(Mandatory)]
    [System.IO.FileInfo]$DiskImage,

    [ValidateSet('install','boot', 'iso')]
    [String]$ImageType = 'install',

    [int]$ImageIndex = 1
  )

  if (Test-Path $DiskImage -PathType Leaf) {
    [DiskImageInfo]$disk_image_info = Open-DiskImage -DiskImage $DiskImage
  }
  elseif (Test-Path $DiskImage -PathType Container) {
    $disk_image_info = [DiskImageInfo]::new($null, $DiskImage)
  }

  if ($disk_image_info.ContentLocation.IsReadOnly) {
    throw "Disk image folder is ReadOnly: $($disk_image_info.ContentLocation)"
  }

  # The following steps do not apply when this command is executed to perform
  # modifications on the iso and not its contained images
  if ($ImageType -in ('install', 'boot')) {
    $image_path = "$($disk_image_info.ContentLocation)\sources\$ImageType.wim"
    if (-not (Test-Path $image_path)) {
      throw "Image file not found at $image_path"
    }

    $windows_image_info = [WindowsImageInfo]::new($image_path)
    $disk_image_info.WindowsImage = $windows_image_info
    $windows_image_info.ImageType = $ImageType

    if ($windows_image_info.File.IsReadOnly) {
      Write-information "Image file was Read Only, attempting to make ReadWrite"
      $windows_image_info.File.IsReadOnly = $false

      if ($windows_image_info.File.IsReadOnly) {
        throw "Failed to change Image File from Read Only"
      }
    }

    $get_windows_image_params = @{
      'ImagePath' = $windows_image_info.File
      'Index' = $ImageIndex
    }
    $image_info = Get-WindowsImage @get_windows_image_params

    if ($image_info) {
      $windows_image_info.ImageInfo = $image_info
      $windows_image_info.ContentLocation = "$env:TEMP\$($windows_image_info.ImageInfo.ImageName)"
    }
    else {
      throw "Failed to get image info from index $ImageIndex in $($windows_image_info.File)"
    }

    if (Test-Path $windows_image_info.ContentLocation) {
      Remove-Item $windows_image_info.ContentLocation -Recurse -Force -Confirm:$false -ErrorAction Stop
    }

    # Directory must exist for the mount command to work
    $null = New-Item -Path $windows_image_info.ContentLocation -ItemType Directory

    if (Test-Path $windows_image_info.ContentLocation) {
      $mount_windowsimage_params = @{
        'Path' = $windows_image_info.ContentLocation
        'ImagePath' = $windows_image_info.File
        'Index' = $windows_image_info.ImageInfo.ImageIndex
      }
      $windows_image_info.Mount = Mount-WindowsImage @mount_windowsimage_params
    }
    else {
      throw "Image mount folder not present at: $($windows_image_info.ContentLocation)"
    }
  }

  $disk_image_info | Write-Output
}