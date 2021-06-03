function Open-DiskImage {
  <#
    .SYNOPSIS
    Get the contents of a ISO file and place them into a read/write folder.

    .DESCRIPTION
    Mounts an ISO file using DISM commands. Copies its contents into a new temp
    folder that is read writable. Returns an object with the location of that
    folder. Will also return the related mount and volume information if
    KeepMounted is specified.

    .PARAMETER DiskImage
    The full path to the location of the DiskImage file to use as source material

    .PARAMETER KeepMounted
    Switch for keeping the DiskImage mounted after its contents have been copied to a
    another folder

    .EXAMPLE
    Open a DiskImage file.

    PS> $DiskImage = C:\Win10.iso
    PS> Open-DiskImage -DiskImage $DiskImage

    .EXAMPLE
    Open an DiskImage file and keep the DiskImage mounted.

    PS> $DiskImage = C:\Win10.iso
    PS> Open-DiskImage -DiskImage $DiskImage -KeepMounted

    .INPUTS
    [String[]]

    .OUTPUTS
    [DiskImageInfo]

    .LINK
    https://docs.microsoft.com/en-us/windows-hardware/manufacture/desktop/dism---deployment-image-servicing-and-management-technical-reference-for-windows
    https://docs.microsoft.com/en-us/powershell/module/dism/?view=win10-ps
    https://docs.microsoft.com/en-us/powershell/module/storage/?view=win10-ps
    https://docs.microsoft.com/en-us/windows-hardware/manufacture/desktop/oscdimg-command-line-options
  #>

  [cmdletbinding()]
  param (
    [Alias('File')]
    [Parameter(Mandatory)]
    [ValidateScript({(Test-Path $_ -PathType leaf) -and (($_ | Split-Path -Leaf).length -lt 125)})]
    [System.IO.FileInfo]$DiskImage,

    [Switch]$KeepMounted
  )

  $disk_image_info = [DiskImageInfo]::new($DiskImage)

  Write-Information "Attempting to mount file $($disk_image_info.File)"
  $disk_image_info.Mount = Mount-DiskImage -ImagePath $disk_image_info.File -PassThru

  if (-not ($disk_image_info.Mount)) {
    throw "Failed to mount $($disk_image_info.File)"
  }

  <# TODO(actodd): Get volume of mount instead of newest volume
    This is a blunt way to try and get the newly created volume.
    Using Mount-DiskImage does not consistantly create a volume that can be
    queried through Get-Volume. I have done a lot of testing and research to
    determine why. Nothing seems to consistently cause it to be queriable.
    This command is consistently working but I have no way to target the volume
    I am looking for from the information returned by Mount-DiskImage. In
    testing, the newest volume was always the last volume.
  #>
  $disk_image_info.Volume = (Get-WmiObject -Class Win32_Volume)[-1]
  if (-not($disk_image_info.Volume)) {
    throw 'Failed to get mounted ISO Volume information'
  }

  $disk_image_info.DiskRoot = "$($disk_image_info.Volume.name)"
  Write-Information "ISO root set to $($disk_image_info.DiskRoot)"

  # TODO(actodd): Replace this blunt edge case handling loop for a more
  # percises set of mount and volume handling code.
  $counter = 0
  while ($true) {
    if (Test-Path -Path $disk_image_info.DiskRoot) {
      Write-Information "Found $($disk_image_info.DiskRoot), continuing."
      break
    }

    if ($counter -eq 5) {
      throw "Failed to find $($disk_image_info.DiskRoot) after $counter intervals."
    }

    if ($VerbosePreference) {
      Write-Verbose "$(Get-PSDrive)"
    }
    Write-Verbose "$($disk_image_info.DiskRoot) not found. Sleeping for 1 second."
    Start-Sleep -Seconds 1
    $counter++
  }

  # TODO(actodd): Use the diskimage CimInstance GUID IS for the temp folder name
  $iso_name = ($disk_image_info.File | Split-Path -Leaf).split('.')[0] -replace  ' ','_'
  $disk_image_info.ContentLocation = "$env:TEMP\$iso_name"

  Write-Information "Checking for the existence of $($disk_image_info.ContentLocation)"
  if (Test-Path $disk_image_info.ContentLocation) {
    Write-Information "Removing pre-existing directory $($disk_image_info.ContentLocation)"
    $null = Remove-Item -Path $disk_image_info.ContentLocation -Recurse -Force -Confirm:$false
    if (Test-Path $disk_image_info.ContentLocation) {
      throw "Failed to remove directory $($disk_image_info.ContentLocation)"
    }
  }

  if (Test-Path $disk_image_info.DiskRoot) {
    Write-Information "Attempting to copy items from $($disk_image_info.DiskRoot) to $($disk_image_info.ContentLocation) recursively"
    $null = Copy-Item -Path $disk_image_info.DiskRoot -Destination $disk_image_info.ContentLocation -Recurse -Force
  }
  else {
    throw "Failed to verify that $($disk_image_info.DiskRoot) exists"
  }

  if ($KeepMounted) {
    Write-Information "Leaving ISO mounted at $($disk_image_info.DiskRoot)"
  }
  else {
    Write-Information "Dismounting ISO from $($disk_image_info.DiskRoot)"
    $null = Dismount-DiskImage -ImagePath $disk_image_info.File
    # TODO(actodd): Implement [DiskImageInfo] update method override for
    # checking if the properties are still valid.
    $disk_image_info.Mount = $null
    $disk_image_info.DiskRoot = $null
    $disk_image_info.Volume =  $null
  }

  $disk_image_info | Write-Output
}