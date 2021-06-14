# TODO (actodd): Change this to Add-ItemToDiskImage and update all references
function Add-FileToDiskImage {
  <#
    .SYNOPSIS
      Add a File or Folder to a Disk Image file or folder

    .DESCRIPTION
      Add a file or folder to a Disk Image file (ISO) or a folder of its
      contents. The item will be added to the path specified with a root of
      the the Disk Image content folder. Copy-Item is used to perform this
      action, so the presence and permisions of the files involved will perform
      the same as if you had specified Copy-Item manually.

    .PARAMETER DiskImage
      The disk image file or folder to add the files to.

    .PARAMETER ImageIndex
      The index id of the image to add the files too. The default value is '1'.

    .PARAMETER DependencyInfoObj
      An Object containing the source and destination information for a
      Dependency that needs loaded into the image.

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

      PS> Copy-FileToDiskImage -DiskImage C:\win10.iso -SourceFile C:\unattend.xml -ImageDestinationPath unattend.xml

    .EXAMPLE
      Copy the file C:\unattend.xml to the "$env:SystemDrive:\Windows\Panther" folder of the disk image win10.iso.
      the image type to be 'Boot' with an index of 2.

      PS> Copy-FileToDiskImage -DiskImage C:\win10.iso -SourceFile C:\unattend.xml -ImageDestinationPath 'Windows\Panther\unattend.xml' -ImageType Boot -ImageIndex 2
  #>

  [cmdletbinding()]
  param (
    [parameter(Mandatory)]
    [ValidateScript({Test-Path $_})]
    [System.IO.FileInfo]$DiskImage,

    [Parameter(
      Mandatory,
      ParameterSetName = "WindowsImageDependenciesInfo"
    )]
    [WindowsImageDependenciesInfo[]]$DependencyInfoObj,

    [Parameter(
      Mandatory,
      ParameterSetName = "System.IO.FileInfo"
    )]
    [ValidateScript({(Test-Path -Path $_)})]
    [System.IO.FileInfo]$SourceFile,

    [Parameter(
      Mandatory,
      ParameterSetName = "System.IO.FileInfo"
    )]
    [ValidateScript({$_ -notmatch '(^.*\:\\|^\\)'})]
    [String]$ImageDestinationPath,

    [int]$ImageIndex = 1,

    # TODO(actodd): Add support for adding files to the DiskImage and not an
    # included WIM file.
    [ValidateSet('boot','install', 'iso')]
    [String]$ImageType = 'install',

    [Switch]$Force
  )

  [system.collections.stack]$cleanup_commands = @()

  try {
    $open_disk_image_environment_params = @{
      'DiskImage' = $DiskImage
      'ImageType' = $ImageType
    }

    if ($ImageType -in ('boot', 'install')) {
      $open_disk_image_environment_params.Add('ImageIndex', $ImageIndex)
    }

    [DiskImageInfo]$disk_image_info = Open_DiskImageEnvironment @open_disk_image_environment_params
    Write-Information 'Register Close_DiskImageEnvironment cleanup command'
    $cleanup_commands.push(@{
        'Scriptblock' = {Close_DiskImageEnvironment -DiskImageInfoObj $args[0] -Force:$args[1]}
        'ArgumentList' = ($disk_image_info,$Force)
      }
    )

    if ($ImageType -in ('boot', 'install')) {
      $root_directory = $disk_image_info.WindowsImage.ContentLocation
    }
    elseif ($ImageType -eq 'iso') {
      $root_directory = $disk_image_info.ContentLocation
    }

    [System.Collections.ArrayList]$dep_pairs = @()
    if ('DependencyInfoObj' -in $PSBoundParameters.keys) {
      foreach ($info_obj in $DependencyInfoObj) {
        $source = $info_obj.SourceLocation
        $dest = Join-Path $root_directory ($info_obj.DestinationLocation)
        $dep_pairs.add(@($source, $dest)) | Out-Null
      }
    } elseif ('SourceFile' -in $PSBoundParameters.keys) {
      $dest = Join-Path $root_directory $ImageDestinationPath
      $dep_pairs.add(@($SourceFile.FullName, $dest)) | Out-Null
    } else {
      throw "Failed to enumerate parameter set name of $($PSCmdlet.ParameterSetName)"
    }

    foreach ($pair in $dep_pairs) {
      # For this to unpack properly the variables must be connected without spaces.
      $source,$dest = $pair
      # TODO (actodd): Split this into two different commands depending on if its a
      # file or folder.
      Write-Information "Attempting to copy $source to $dest"
      if (Test-Path $source) {
        Copy-Item -Path $source -Destination $dest -Force:$Force -Confirm:$false -Recurse
      } else {
        throw "Failed to find item to copy at $source"
      }

      if (-not(Test-Path $dest)) {
        throw "Failed to copy $source to $dest"
      }
    }
  }
  finally {
    foreach ($c in $cleanup_commands) {
      try {
        Invoke-Command @c
      } catch {
        Write-Error $_
      }
    }
  }
}