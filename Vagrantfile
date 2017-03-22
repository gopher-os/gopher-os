# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.define "gopheros-build" do |v|
  end

  config.vm.provider "virtualbox" do |vb|
    vb.customize ["modifyvm", :id, "--usb", "on"]
    vb.customize ["modifyvm", :id, "--usbehci", "off"]
  end

  config.vm.box = "minimal/trusty64"

  config.vm.synced_folder "./", "/home/vagrant/workspace/src/github.com/achilleasa/gopher-os"

  config.vm.provision "shell", inline: <<-SHELL
    apt-get update
    apt-get install -y nasm gccgo xorriso
    [ ! -d "/usr/local/go" ] && wget -qO- https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz | tar xz -C /usr/local
    echo "export GOROOT=/usr/local/go" > /etc/profile.d/go.sh
    echo "export GOBIN=/usr/local/go/bin" >> /etc/profile.d/go.sh
    echo "export GOPATH=/home/vagrant/workspace" >> /etc/profile.d/go.sh
    echo "export PATH=$PATH:/usr/local/go/bin" >> /etc/profile.d/go.sh
  SHELL
end
