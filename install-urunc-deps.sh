#!/bin/bash
set -e

echo "=================================================="
echo "  urunc Development Environment Installation"
echo "  for Ubuntu 24.04 LTS in WSL2"
echo "=================================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

print_step() {
    echo -e "${BLUE}==>${NC} ${GREEN}$1${NC}"
}

print_error() {
    echo -e "${RED}ERROR:${NC} $1"
}

# Step 1: Update system and install prerequisites
print_step "Step 1: Updating system and installing prerequisites..."
sudo apt update
sudo apt install -y wget curl bc build-essential libseccomp-dev git

# Step 2: Install runc
print_step "Step 2: Installing runc..."
RUNC_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/opencontainers/runc/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
echo "  Installing runc version: $RUNC_VERSION"
wget -q https://github.com/opencontainers/runc/releases/download/v$RUNC_VERSION/runc.$(dpkg --print-architecture)
sudo install -m 755 runc.$(dpkg --print-architecture) /usr/local/sbin/runc
rm -f ./runc.$(dpkg --print-architecture)
runc --version

# Step 3: Install containerd
print_step "Step 3: Installing containerd..."
CONTAINERD_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/containerd/containerd/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
echo "  Installing containerd version: $CONTAINERD_VERSION"
wget -q https://github.com/containerd/containerd/releases/download/v$CONTAINERD_VERSION/containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo tar Cxzvf /usr/local containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz
rm -f containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz

# Install containerd service
print_step "Step 3a: Setting up containerd service..."
wget -q https://raw.githubusercontent.com/containerd/containerd/v$CONTAINERD_VERSION/containerd.service
sudo rm -f /lib/systemd/system/containerd.service
sudo mv containerd.service /lib/systemd/system/containerd.service

# Note: systemd doesn't work normally in WSL2, but we'll set it up anyway
# You may need to start containerd manually with: sudo containerd &

# Step 4: Install CNI plugins
print_step "Step 4: Installing CNI plugins..."
CNI_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/containernetworking/plugins/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
echo "  Installing CNI plugins version: $CNI_VERSION"
wget -q https://github.com/containernetworking/plugins/releases/download/v$CNI_VERSION/cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz
sudo mkdir -p /opt/cni/bin
sudo tar Cxzvf /opt/cni/bin cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz
rm -f cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz

# Step 5: Configure containerd
print_step "Step 5: Configuring containerd..."
sudo mkdir -p /etc/containerd/
if [ -f /etc/containerd/config.toml ]; then
    sudo mv /etc/containerd/config.toml /etc/containerd/config.toml.bak
fi
sudo containerd config default | sudo tee /etc/containerd/config.toml > /dev/null

# Configure devmapper (we'll use blockfile for WSL2 as it's simpler)
print_step "Step 5a: Setting up blockfile snapshotter..."
sudo mkdir -p /opt/containerd/blockfile
sudo dd if=/dev/zero of=/opt/containerd/blockfile/scratch bs=1M count=500
sudo mkfs.ext4 /opt/containerd/blockfile/scratch -F
sudo chown -R root:root /opt/containerd/blockfile

# Add blockfile configuration to containerd config
print_step "Step 5b: Updating containerd configuration for blockfile..."
cat << 'EOF' | sudo tee -a /etc/containerd/config.toml > /dev/null

[plugins.'io.containerd.snapshotter.v1.blockfile']
  fs_type = "ext4"
  mount_options = []
  recreate_scratch = true
  root_path = "/var/lib/containerd/io.containerd.snapshotter.v1.blockfile"
  scratch_file = "/opt/containerd/blockfile/scratch"
  supported_platforms = ["linux/amd64"]
EOF

# Step 6: Install nerdctl
print_step "Step 6: Installing nerdctl..."
NERDCTL_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/containerd/nerdctl/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
echo "  Installing nerdctl version: $NERDCTL_VERSION"
wget -q https://github.com/containerd/nerdctl/releases/download/v$NERDCTL_VERSION/nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo tar Cxzvf /usr/local/bin nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
rm -f nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
nerdctl --version

# Step 7: Install crictl
print_step "Step 7: Installing crictl..."
VERSION="v1.30.0"
echo "  Installing crictl version: $VERSION"
wget -q https://github.com/kubernetes-sigs/cri-tools/releases/download/$VERSION/crictl-$VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo tar zxvf crictl-$VERSION-linux-$(dpkg --print-architecture).tar.gz -C /usr/local/bin
rm -f crictl-$VERSION-linux-$(dpkg --print-architecture).tar.gz

# Configure crictl endpoints
print_step "Step 7a: Configuring crictl endpoints..."
sudo tee /etc/crictl.yaml > /dev/null <<'EOT'
runtime-endpoint: unix:///run/containerd/containerd.sock
image-endpoint: unix:///run/containerd/containerd.sock
timeout: 20
EOT
crictl --version

# Step 8: Install Go (if not already installed from Windows)
print_step "Step 8: Checking Go installation..."
if ! command -v go &> /dev/null; then
    print_step "Installing Go 1.24.6..."
    wget -q https://go.dev/dl/go1.24.6.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go1.24.6.linux-amd64.tar.gz
    rm go1.24.6.linux-amd64.tar.gz
    
    # Add to PATH
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
        export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
    fi
else
    echo "  Go already installed: $(go version)"
fi

# Step 9: Install QEMU (optional but recommended for testing)
print_step "Step 9: Installing QEMU (optional)..."
read -p "Do you want to install QEMU? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    sudo apt install -y qemu-system
    qemu-system-x86_64 --version
fi

echo ""
echo "=================================================="
echo -e "${GREEN}Installation completed successfully!${NC}"
echo "=================================================="
echo ""
echo "Next steps:"
echo "1. Start containerd: sudo containerd &"
echo "   (In WSL2, systemd may not work, so start manually)"
echo ""
echo "2. Navigate to urunc directory and build:"
echo "   cd /mnt/c/Users/Mradul/urunc"
echo "   make && sudo make install"
echo ""
echo "3. Run tests:"
echo "   make unittest"
echo ""
echo "Note: For e2e tests, you'll need hypervisors (QEMU, Firecracker, Solo5)"
echo ""
