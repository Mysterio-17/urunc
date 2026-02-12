# LFX Mentorship Proposal: Read-Only Snapshot Implementation for urunc

## Applicant Information

**Name:** Mradul  
**GitHub:** @Mysterio-17  
**Project:** urunc - Use a RO Snapshot of Container to Retrieve Unikernel Binary  
**Issue:** [#43](https://github.com/urunc-dev/urunc/issues/43)  
**Commitment:** 40 hours per week for 12 weeks (full-time dedication)

---

## Project Overview

urunc is a CNCF Sandbox project that reimagines container runtimes for the unikernel ecosystem. While traditional runtimes like runc spawn Linux processes, urunc takes a different path - it boots unikernels inside Virtual Machine Monitors (VMMs) or sandbox monitors, treating the VMM process itself as the container. This design eliminates auxiliary processes while giving unikernels the full isolation benefits of virtualization, all managed through familiar tools like Docker, nerdctl, or Kubernetes.

The runtime integrates seamlessly into the containerd ecosystem through a custom shim (`containerd-shim-urunc-v2`), supporting five unikernel frameworks - Unikraft, Rumprun, MirageOS, Mewz, and Linux minimal VMs - across VMMs including QEMU, Firecracker, Solo5-hvt, and Solo5-spt. One of urunc's strengths is its flexible rootfs handling: it can work with initrd, block devices, 9pfs, or virtiofs depending on the unikernel and VMM combination.

For unikernels that support block devices natively (like Rumprun and Linux minimal VMs), urunc can leverage block-based snapshotters like devmapper to pass the container's rootfs directly as a block device - avoiding filesystem conversion overhead entirely. However, this creates a challenge: the unikernel binary, initrd, and configuration files (urunc.json) reside inside that same rootfs, and urunc needs to read them *before* handing over the block device to the VMM. The current workaround copies these files to `/run`, which accumulates storage overhead with each container instance.

---

## Problem Statement

### Current Implementation Analysis

When urunc uses a block-based snapshotter like devmapper, it gains the ability to pass the container's rootfs directly as a block device to the unikernel. However, this workflow has a critical inefficiency: urunc must first extract the kernel binary, initrd, and configuration files before it can release the rootfs to the VMM.

Looking at the code in `pkg/unikontainers/block.go`, I traced through the block device preparation logic and found the problematic function that handles this file extraction:

**[DIAGRAM 1: Insert here - "Current Block-Based Container Flow"]**
*Prompt for diagram: Create a flowchart showing: Container Image → devmapper snapshot → mount rootfs → copy kernel/initrd/urunc.json to /run → unmount rootfs → pass block device to VMM → unikernel boots. Highlight the "copy files" step in red as the bottleneck.*

The `extractFilesFromBlock()` function is where this happens:

**[CODE SNIPPET 1]**  
*File: `pkg/unikontainers/block.go`*  
*Lines: 82-112*  
*Function: `extractFilesFromBlock()`*  
*(Take screenshot of this function - note the FIXME comment about filling up /run)*

The FIXME comment in the code itself acknowledges the problem - every container instance leaves behind copies of potentially large files (kernel binaries can be several megabytes) in `/run`. Over time, this accumulates and can fill up the tmpfs where `/run` typically resides.

### Function Call Chain (Cascade Effect)

The file extraction is called from `prepareDMAsBlock()` function, which then gets invoked by `handleCntrRootfsAsBlock()`. This chain means the copy operation is tightly coupled with the block device preparation logic. Any solution needs to untangle this dependency.

**[CODE SNIPPET 2]**  
*File: `pkg/unikontainers/block.go`*  
*Lines: 172-202*  
*Function: `handleCntrRootfsAsBlock()`*  
*(Take screenshot - shows how block rootfs is currently prepared)*

### Root Cause

The fundamental issue is that urunc currently has no way to maintain two simultaneous views of the same container rootfs - one for reading configuration files, and another (the original block device) for passing to the VMM. The copy operation exists only because we need to read files from a filesystem that we're about to unmount.

---

## Proposed Solution

### Core Idea: View Snapshots via containerd API

Containerd's snapshotter interface supports creating "view" snapshots - read-only references to existing snapshot layers. A view snapshot doesn't duplicate data; it simply provides another mount point that redirects reads to the underlying layers. This is exactly what urunc needs.

Instead of:
1. Mount container rootfs
2. Copy kernel, initrd, urunc.json to /run
3. Unmount rootfs
4. Pass block device to VMM

We would:
1. Request a view snapshot from containerd
2. Mount view snapshot (read-only)
3. Read kernel, initrd, urunc.json directly from view mount
4. Pass original block device to VMM
5. Cleanup view snapshot after VMM starts

**[DIAGRAM 2: Insert here - "Proposed RO Snapshot Flow"]**
*Prompt for diagram: Create a flowchart showing: Container Image → devmapper snapshot (splits into two paths) → Path 1: "view" RO snapshot → mount → read kernel/initrd/config → Path 2: original block device → pass to VMM → both paths merge at "unikernel boots". Show the view snapshot with a dotted line to indicate it shares data with original.*

### Integration Point: The Upcoming urunc Shim

Based on the discussion with @cmainas, urunc is developing a new shim that will have direct access to containerd's snapshotter API. This is crucial because currently urunc runs as just an OCI runtime and doesn't have a communication channel back to containerd.

The containerd snapshotter service exposes these relevant methods:
- `View()` - Create a read-only snapshot from an existing snapshot
- `Mounts()` - Get mount information for a snapshot
- `Remove()` - Clean up a snapshot

### Proposed Architecture

To implement this solution cleanly, I'm proposing a new package `pkg/unikontainers/snapshot/` dedicated to managing view snapshot operations. This keeps the snapshot logic modular and separate from the existing block device handling code:

**[CODE SNIPPET 3]**  
*File: `pkg/unikontainers/snapshot/snapshot.go`*  
*(This is a NEW file I created - take screenshot of the entire file)*  
*Shows: ViewManager struct with CreateView() and CleanupView() methods*

### Modified Block Handling

The `handleCntrRootfsAsBlock()` function would be modified to use view snapshots when available:

**[CODE SNIPPET 4]**  
*File: `pkg/unikontainers/block_view.go`*  
*(This is a NEW file I created - take screenshot of the entire file)*  
*Shows: handleCntrRootfsAsBlockWithView() function with fallback logic*

---

## Technical Deep Dive

### Snapshotter Compatibility

One of my concerns was whether this approach works across different snapshotters. After reading the containerd documentation and the discussion comments, I'm confident it does. The containerd snapshotter API abstracts the underlying implementation - whether it's devmapper, blockfile, overlaybd, or any other snapshotter, the `View()` call works the same way.

For devmapper specifically, a view snapshot creates a thin device that shares data blocks with the parent. There's minimal overhead because copy-on-write only happens on writes, and view snapshots are read-only by definition.

### Shim Communication Path

The current `containerd-shim-urunc-v2` uses containerd's standard shim manager. The new shim will need to:
1. Establish a gRPC connection to containerd's snapshots service
2. Pass the snapshotter client to urunc's execution context
3. Handle snapshot cleanup even if the container fails

### Edge Cases and Failure Handling

What happens if snapshot creation fails? The implementation should gracefully fallback to the current copy-based approach. This ensures backward compatibility and robustness. You can see this fallback logic in the `handleCntrRootfsAsBlockWithView()` function where we catch the error and call the original `handleCntrRootfsAsBlock()` instead.

---

## Alternative Approach Considered

### Direct Filesystem Parsing (Memory-Mapped Access)

Before settling on the view snapshot approach, I explored the idea of directly reading files from the block device without mounting it at all. The concept was to memory-map the block device and parse the filesystem structure manually to locate and extract the kernel binary, initrd, and urunc.json files.

This approach had one attractive property: zero mount overhead. We'd completely skip the mount/unmount cycle, which in theory would be faster. The implementation would involve:
- Opening the block device directly
- Parsing the filesystem superblock to understand the layout
- Traversing directory entries to locate target files
- Reading file contents directly from the mapped regions

However, after deeper analysis, this approach has significant drawbacks that make it impractical:

**Filesystem-specific implementation:** Each filesystem (ext4, xfs, btrfs, etc.) has a completely different on-disk format. We'd need separate parsing logic for every filesystem type that users might use with devmapper. This is a massive implementation burden.

**Maintenance nightmare:** Filesystem formats evolve. Even within ext4, there are multiple feature flags and layout variations. Keeping up with these changes would require ongoing effort and deep filesystem expertise.

**High risk of subtle bugs:** Filesystem parsing is notoriously tricky. One off-by-one error in block calculation could lead to reading garbage data or, worse, corrupting the understanding of the filesystem state.

**Reinventing the wheel:** The Linux kernel already has robust, battle-tested filesystem drivers. The view snapshot approach leverages this existing infrastructure through containerd's snapshotter API, rather than reimplementing it poorly.

The view snapshot solution is cleaner, more maintainable, and works across all filesystems without any filesystem-specific code. It's the right architectural choice.

---

## CI/CD and Testing Integration

### GitHub Actions Workflow

The implementation would include automated testing through GitHub Actions. The workflow covers:

- **Unit tests** for the snapshot package, triggered on changes to relevant files
- **Integration tests** with devmapper snapshotter on a real thinpool setup
- **Verification step** to confirm `/run` isn't accumulating kernel copies

**[CODE SNIPPET 5]**  
*File: `.github/workflows/test-view-snapshot.yml`*  
*(This is a NEW file I created - take screenshot of the workflow)*

## Challenges and Limitations

### Project Challenges

**Shim Development Dependency:** This feature depends on the new urunc shim that provides access to containerd's snapshotter API. Close coordination with maintainers will be needed to design the integration points properly.

**Snapshot Lifecycle Management:** The view snapshot must stay mounted while urunc reads kernel files, but should be cleaned up promptly to avoid leaks. Proper error handling is needed to prevent orphaned snapshots when VMM startup fails.

**Cross-Snapshotter Compatibility:** While containerd's API abstracts snapshotter details, subtle behavioral differences between devmapper and blockfile may surface during testing.

### Setup Difficulties I Encountered

**devmapper thinpool requires specific tools:** The `dm_create.sh` script assumes LVM tools are available. On Ubuntu, `thin-provisioning-tools` must be installed separately, otherwise thinpool creation silently fails.

**KVM requires nested virtualization:** Running urunc inside a VM requires nested virtualization enabled. Without it, QEMU falls back to software emulation which is significantly slower.

### Technical Limitations

**Snapshotter Dependency:** This solution only works with block-based snapshotters. The implementation includes fallback logic for non-block snapshotters.

**Cleanup Timing:** View snapshots must be cleaned up after VMM starts. The solution synchronizes cleanup with VMM startup confirmation.

---

## 12-Week Implementation Roadmap

**Weeks 1-2:**
- Research containerd snapshotter API (`PrepareSnapshot`, `ViewSnapshot`, `RemoveSnapshot`)
- Set up devmapper/blockfile environment locally
- Analyze urunc shim architecture and coordinate with maintainers
- Create design document and present to mentors for feedback

**Week 3:**
- Implement `pkg/unikontainers/snapshot/` package with ViewManager struct
- Write unit tests with mocked containerd client
- Share initial implementation with mentors for early code review

**Week 4:**
- Integrate snapshot package with block.go
- Modify `handleCntrRootfsAsBlock()` to use view snapshots
- Implement fallback logic for graceful degradation
- Weekly sync with mentors to discuss integration challenges

**Week 5:**
- Wire up snapshotter client through the shim
- Handle snapshot lifecycle (creation, mounting, cleanup)
- Initial integration testing with devmapper
- Open draft PR for mentor visibility and async feedback

**Weeks 6-7:**
- Comprehensive unit and integration tests with devmapper/blockfile
- End-to-end testing with Rumprun and Linux minimal VMs
- Performance benchmarking (startup time, storage overhead)
- Demo working implementation to mentors

**Week 8:**
- Edge case testing (concurrent starts, cleanup during failures)
- Fix bugs identified during testing
- Code cleanup and refactoring based on mentor feedback

**Weeks 9-10:**
- Update documentation with snapshotter setup best practices
- Performance evaluation report with before/after benchmarks
- Prepare pull request with comprehensive description

**Weeks 11-12:**
- Iterate on PR based on maintainer feedback
- Ensure CI passes on all supported platforms
- Final documentation polish and merge preparation

---

## Beyond the 12 Weeks

This project is my entry point into the urunc community, not my exit. After completing this feature, here's what I'm looking forward to:

1. **Contribute to the new shim development** - since this feature requires the shim, I'll naturally be familiar with its internals and can help with ongoing improvements and new features.

2. **Take on other urunc issues** - I've noticed interesting work around WASM sandbox integration and improved Kubernetes support that I'd like to explore after this mentorship.

3. **Guide new contributors** - having gone through the onboarding and setup process myself, I want to help smooth the path for future contributors by improving documentation and being available on Slack to answer questions.

4. **Grow with the project** - I see this mentorship as the first step in a longer journey with urunc, with the goal of becoming a trusted community member who can help review PRs, triage issues, and mentor others joining the project.

---

## Commitment and Communication

I am fully committed to dedicating **40 hours per week** to this project throughout the 12-week mentorship period. This is my primary focus, and I have structured my schedule to ensure uninterrupted time for deep work on urunc.

**Slack:** I'm already active on the #urunc channel in CNCF Slack and will use it for daily progress updates, quick questions, and engaging with the broader community.

**Weekly Sync Meetings:** I'd like to establish a regular weekly 30-45 minute sync with my mentor(s) to review progress, discuss blockers, and plan upcoming work. I'm flexible on timing and platform.

**GitHub Discussions:** All code-related discussions will happen through PR comments and issue threads. I'll open draft PRs early to get async feedback and ensure detailed commit messages for clarity.

