package integration

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Upgrade", func() {
	Describe("Upgrading a cluster using offline mode", func() {
		Describe("From KET version v1.4.1", func() {
			BeforeEach(func() {
				dir := setupTestWorkingDirWithVersion("v1.4.1")
				os.Chdir(dir)
			})
			Context("Using a minikube layout", func() {
				Context("Using CentOS 7", func() {
					ItOnAWS("should be upgraded [slow] [upgrade]", func(aws infrastructureProvisioner) {
						WithMiniInfrastructure(CentOS7, aws, func(node NodeDeets, sshKey string) {
							installAndUpgradeMinikube(node, sshKey, false)
						})
					})
				})

				Context("Using RedHat 7", func() {
					ItOnAWS("should be upgraded [slow] [upgrade]", func(aws infrastructureProvisioner) {
						WithMiniInfrastructure(RedHat7, aws, func(node NodeDeets, sshKey string) {
							installAndUpgradeMinikube(node, sshKey, false)
						})
					})
				})
			})

			// This spec will be used for testing non-destructive kismatic features on
			// an upgraded cluster.
			// This spec is open to modification when new assertions have to be made.
			Context("Using a skunkworks cluster", func() {
				ItOnAWS("should result in an upgraded cluster [slow] [upgrade]", func(aws infrastructureProvisioner) {
					WithInfrastructureAndDNS(NodeCount{Etcd: 3, Master: 2, Worker: 3, Ingress: 2, Storage: 2}, Ubuntu1604LTS, aws, func(nodes provisionedNodes, sshKey string) {
						// reserve one of the workers for the add-worker test
						allWorkers := nodes.worker
						nodes.worker = allWorkers[0 : len(nodes.worker)-1]

						// Standup cluster with previous version
						opts := installOptions{}
						err := installKismatic(nodes, opts, sshKey)
						FailIfError(err)

						// Extract current version of kismatic
						extractCurrentKismaticInstaller()

						// Perform upgrade
						upgradeCluster(false)

						sub := SubDescribe("Using an upgraded cluster")
						defer sub.Check()

						sub.It("should have working storage volumes", func() error {
							return testStatefulWorkload(nodes, sshKey)
						})

						sub.It("should allow adding a worker node", func() error {
							newWorker := allWorkers[len(allWorkers)-1]
							return addWorkerToCluster(newWorker)
						})

						sub.It("should be able to deploy a workload with ingress", func() error {
							return verifyIngressNodes(nodes.master[0], nodes.ingress, sshKey)
						})

						// Use master[0] public IP
						sub.It("should have an accessible dashboard", func() error {
							return canAccessDashboard(fmt.Sprintf("https://admin:abbazabba@%s:6443/ui", nodes.master[0].PublicIP))
						})

						sub.It("should respect network policies", func() error {
							return verifyNetworkPolicy(nodes.master[0], sshKey)
						})

						// This test should always be last
						sub.It("should still be a highly available cluster after upgrade", func() error {
							By("Removing a Kubernetes master node")
							if err = aws.TerminateNode(nodes.master[0]); err != nil {
								return fmt.Errorf("could not remove node: %v", err)
							}
							By("Re-running Kuberang")
							if err = runViaSSH([]string{"sudo kuberang"}, []NodeDeets{nodes.master[1]}, sshKey, 5*time.Minute); err != nil {
								return err
							}
							return nil
						})
					})
				})
			})

			Context("Using a cluster that has no internet access [slow] [upgrade]", func() {
				ItOnAWS("should result in an upgraded cluster", func(aws infrastructureProvisioner) {
					distro := CentOS7
					WithInfrastructure(NodeCount{Etcd: 3, Master: 1, Worker: 1}, distro, aws, func(nodes provisionedNodes, sshKey string) {
						// Standup cluster with previous version
						opts := installOptions{
							disconnectedInstallation:    false, // we want KET to install the packages, so let it use the package repo
							modifyHostsFiles:            true,
							autoConfigureDockerRegistry: true,
						}
						err := installKismatic(nodes, opts, sshKey)
						FailIfError(err)

						// Extract current version of kismatic
						extractCurrentKismaticInstaller()

						// Remove old packages
						By("Removing old packages")
						RemoveKismaticPackages()

						// Cleanup old cluster file and create a new one
						By("Recreating kismatic-testing.yaml file")
						err = os.Remove("kismatic-testing.yaml")
						FailIfError(err)
						opts = installOptions{
							disablePackageInstallation:  true,
							disconnectedInstallation:    true,
							modifyHostsFiles:            true,
							autoConfigureDockerRegistry: true,
						}
						writePlanFile(buildPlan(nodes, opts, sshKey))

						// Manually install the new packages
						InstallKismaticPackages(nodes, distro, sshKey, true)

						// Lock down internet access
						err = disableInternetAccess(nodes.allNodes(), sshKey)
						FailIfError(err)

						// Confirm there is not internet
						if err := verifyNoInternetAccess(nodes.allNodes(), sshKey); err == nil {
							Fail("was able to ping google with outgoing connections blocked")
						}

						// Perform upgrade
						upgradeCluster(false)
					})
				})
			})
		})

		Describe("From KET version v1.4.0", func() {
			BeforeEach(func() {
				dir := setupTestWorkingDirWithVersion("v1.4.0")
				os.Chdir(dir)
			})
			Context("Using a minikube layout", func() {
				Context("Using RHEL 7", func() {
					ItOnAWS("should be upgraded [slow] [upgrade]", func(aws infrastructureProvisioner) {
						WithMiniInfrastructure(RedHat7, aws, func(node NodeDeets, sshKey string) {
							installAndUpgradeMinikube(node, sshKey, false)
						})
					})
				})
				Context("Using Ubuntu 16.04", func() {
					ItOnAWS("should be upgraded [slow] [upgrade]", func(aws infrastructureProvisioner) {
						WithMiniInfrastructure(Ubuntu1604LTS, aws, func(node NodeDeets, sshKey string) {
							installAndUpgradeMinikube(node, sshKey, false)
						})
					})
				})

			})
		})
	})

	Describe("Upgrading a cluster using online mode", func() {
		Describe("From KET version v1.4.1", func() {
			BeforeEach(func() {
				dir := setupTestWorkingDirWithVersion("v1.4.1")
				os.Chdir(dir)
			})
			Context("Using a minikube layout", func() {
				Context("Using CentOS 7", func() {
					ItOnAWS("should be upgraded [slow] [upgrade]", func(aws infrastructureProvisioner) {
						WithMiniInfrastructure(CentOS7, aws, func(node NodeDeets, sshKey string) {
							installAndUpgradeMinikube(node, sshKey, true)
						})
					})
				})

				Context("Using RedHat 7", func() {
					ItOnAWS("should be upgraded [slow] [upgrade]", func(aws infrastructureProvisioner) {
						WithMiniInfrastructure(RedHat7, aws, func(node NodeDeets, sshKey string) {
							installAndUpgradeMinikube(node, sshKey, true)
						})
					})
				})
			})

			// This spec will be used for testing non-destructive kismatic features on
			// an upgraded cluster.
			// This spec is open to modification when new assertions have to be made.
			Context("Using a skunkworks cluster", func() {
				ItOnAWS("should result in an upgraded cluster [slow] [upgrade]", func(aws infrastructureProvisioner) {
					WithInfrastructureAndDNS(NodeCount{Etcd: 3, Master: 2, Worker: 3, Ingress: 2, Storage: 2}, Ubuntu1604LTS, aws, func(nodes provisionedNodes, sshKey string) {
						// reserve one of the workers for the add-worker test
						allWorkers := nodes.worker
						nodes.worker = allWorkers[0 : len(nodes.worker)-1]

						// Standup cluster with previous version
						opts := installOptions{}
						err := installKismatic(nodes, opts, sshKey)
						FailIfError(err)

						// Extract current version of kismatic
						extractCurrentKismaticInstaller()

						// Perform upgrade
						upgradeCluster(true)

						sub := SubDescribe("Using an upgraded cluster")
						defer sub.Check()

						sub.It("should have working storage volumes", func() error {
							return testStatefulWorkload(nodes, sshKey)
						})

						sub.It("should allow adding a worker node", func() error {
							newWorker := allWorkers[len(allWorkers)-1]
							return addWorkerToCluster(newWorker)
						})

						sub.It("should be able to deploy a workload with ingress", func() error {
							return verifyIngressNodes(nodes.master[0], nodes.ingress, sshKey)
						})

						// Use master[0] public IP
						sub.It("should have an accessible dashboard", func() error {
							return canAccessDashboard(fmt.Sprintf("https://admin:abbazabba@%s:6443/ui", nodes.master[0].PublicIP))
						})

						sub.It("should respect network policies", func() error {
							return verifyNetworkPolicy(nodes.master[0], sshKey)
						})

						// This test should always be last
						sub.It("should still be a highly available cluster after upgrade", func() error {
							By("Removing a Kubernetes master node")
							if err = aws.TerminateNode(nodes.master[0]); err != nil {
								return fmt.Errorf("could not remove node: %v", err)
							}
							By("Re-running Kuberang")
							if err = runViaSSH([]string{"sudo kuberang"}, []NodeDeets{nodes.master[1]}, sshKey, 5*time.Minute); err != nil {
								return err
							}
							return nil
						})
					})
				})
			})

			Context("Using a cluster that has no internet access [slow] [upgrade]", func() {
				ItOnAWS("should result in an upgraded cluster", func(aws infrastructureProvisioner) {
					distro := CentOS7
					WithInfrastructure(NodeCount{Etcd: 3, Master: 1, Worker: 1}, distro, aws, func(nodes provisionedNodes, sshKey string) {
						// Standup cluster with previous version
						opts := installOptions{
							disconnectedInstallation:    false, // we want KET to install the packages, so let it use the package repo
							modifyHostsFiles:            true,
							autoConfigureDockerRegistry: true,
						}
						err := installKismatic(nodes, opts, sshKey)
						FailIfError(err)

						// Extract current version of kismatic
						extractCurrentKismaticInstaller()

						// Remove old packages
						By("Removing old packages")
						RemoveKismaticPackages()

						// Cleanup old cluster file and create a new one
						By("Recreating kismatic-testing.yaml file")
						err = os.Remove("kismatic-testing.yaml")
						FailIfError(err)
						opts = installOptions{
							disablePackageInstallation:  true,
							disconnectedInstallation:    true,
							modifyHostsFiles:            true,
							autoConfigureDockerRegistry: true,
						}
						writePlanFile(buildPlan(nodes, opts, sshKey))

						// Manually install the new packages
						InstallKismaticPackages(nodes, distro, sshKey, true)

						// Lock down internet access
						err = disableInternetAccess(nodes.allNodes(), sshKey)
						FailIfError(err)

						// Confirm there is not internet
						if err := verifyNoInternetAccess(nodes.allNodes(), sshKey); err == nil {
							Fail("was able to ping google with outgoing connections blocked")
						}

						// Perform upgrade
						upgradeCluster(true)
					})
				})
			})
		})

		Describe("From KET version v1.4.0", func() {
			BeforeEach(func() {
				dir := setupTestWorkingDirWithVersion("v1.4.0")
				os.Chdir(dir)
			})
			Context("Using a minikube layout", func() {
				Context("Using RHEL 7", func() {
					ItOnAWS("should be upgraded [slow] [upgrade]", func(aws infrastructureProvisioner) {
						WithMiniInfrastructure(RedHat7, aws, func(node NodeDeets, sshKey string) {
							installAndUpgradeMinikube(node, sshKey, true)
						})
					})
				})
				Context("Using Ubuntu 16.04", func() {
					ItOnAWS("should be upgraded [slow] [upgrade]", func(aws infrastructureProvisioner) {
						WithMiniInfrastructure(Ubuntu1604LTS, aws, func(node NodeDeets, sshKey string) {
							installAndUpgradeMinikube(node, sshKey, true)
						})
					})
				})

			})
		})
	})
})

func installAndUpgradeMinikube(node NodeDeets, sshKey string, online bool) {
	// Install previous version cluster
	err := installKismaticMini(node, sshKey)
	FailIfError(err)
	extractCurrentKismaticInstaller()
	upgradeCluster(online)
}

func installV130AndUpgradeMinikube(node NodeDeets, sshKey string, online bool) {
	// Install previous version cluster
	sshUser := node.SSHUser
	plan := PlanAWS{
		Etcd:                []NodeDeets{node},
		Master:              []NodeDeets{node},
		Worker:              []NodeDeets{node},
		Ingress:             []NodeDeets{node},
		Storage:             []NodeDeets{node},
		MasterNodeFQDN:      node.PublicIP,
		MasterNodeShortName: node.PublicIP,
		SSHKeyFile:          sshKey,
		SSHUser:             sshUser,
		// Using CIDR to pass tests on KET v1.3.x
		// These versions used a version of kuberang with a bad test
		ServiceCIDR: "172.17.0.0/16",
	}
	err := installKismaticWithPlan(plan, sshKey)
	FailIfError(err)
	extractCurrentKismaticInstaller()
	upgradeCluster(online)
}
func extractCurrentKismaticInstaller() {
	// Extract current version of kismatic
	pwd, err := os.Getwd()
	FailIfError(err)
	err = extractCurrentKismatic(pwd)
	FailIfError(err)
}
func upgradeCluster(online bool) {
	// Perform upgrade
	cmd := exec.Command("./kismatic", "upgrade", "offline", "-f", "kismatic-testing.yaml")
	if online {
		cmd = exec.Command("./kismatic", "upgrade", "online", "-f", "kismatic-testing.yaml", "--ignore-safety-checks")
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Println("Running diagnostics command")
		// run diagnostics on error
		diagsCmd := exec.Command("./kismatic", "diagnose", "-f", "kismatic-testing.yaml")
		diagsCmd.Stdout = os.Stdout
		diagsCmd.Stderr = os.Stderr
		if errDiags := diagsCmd.Run(); errDiags != nil {
			fmt.Printf("ERROR: error running diagnose command: %v", errDiags)
		}
		FailIfError(err)
	}

	assertClusterVersionIsCurrent()
}
