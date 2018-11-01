#!/usr/bin/env groovy

@NonCPS
static def summarizeBuild(b) {
    b.changeSets.collect { cs ->
        /Changes: / + cs.collect { entry ->
            / — ${entry.msg} by ${entry.author.fullName} /
        }.join('\n')
    }.join('\n')
}

def srcVersion = null
def publishVersion = null
def gitCommit = null
def gitBranch = null
def releaseBranches = ['master', 'release/.*']
def releaseBuild = false
def quickBuild = false
def gitTaggerEmail = 'mvasilev@kublr.com'
def gitTaggerName = 'Maksim Vasilev'

podTemplate(
  label: 'kublrslave',
  containers: [
  	        containerTemplate(name: 'jnlp',  image: 'jenkinsci/jnlp-slave', args: '${computer.jnlpmac} ${computer.name}'),
                containerTemplate(name: 'slave', ttyEnabled: true, command: 'cat', args: '-v',
                        image: 'nexus.build.svc.cluster.local:5000/jenkinsci/jnlp-slave-ext:0.1.15')
  ]) {
  node('kublrslave') {
    String buildStatus = 'Success'
    def APP_DIR = "${HOME}/go/src/github.com/kublr/terraform-provider-vcd"
    try {
      sh "mkdir -p ${APP_DIR}"      
      dir ("${APP_DIR}"){
	checkout scm
	stage('build-publish') {	  
	  // get current version from the source code
	  srcVersion = sh returnStdout: true, script: '. ./main.properties; echo -n ${COMPONENT_VERSION}'
	  srcVersion = srcVersion.trim()
	  // fail if version is not found for any reason
	  if (srcVersion == "") { error "Component version is not specified in main.properties" }
	  // get git commit hash
	  gitCommit = sh returnStdout: true, script: 'git rev-parse HEAD'
	  gitCommit = gitCommit.trim()
	  // git branch name is taken from an env var for multi-branch pipeline project, or from git for other projects
	  gitBranch = sh returnStdout: true, script: 'git rev-parse --abbrev-ref HEAD'
	  gitBranch = gitBranch.trim()
	  // branch qualifier for use in tag
	  def branchQual = gitBranch.replaceAll('[^a-zA-Z0-9]', '_')
	  branchQual = branchQual.length() <= 32 ? branchQual : branchQual.substring(branchQual.length() - 32, branchQual.length())
	  // tag names are generated in different ways for release and non-release branches
	  releaseBuild = (gitBranch in releaseBranches)
	  // we need only release builds for the project
	  publishVersion = releaseBuild ? srcVersion : "${srcVersion}-${branchQual}-${BUILD_NUMBER}"
	  sh "echo ${publishVersion} > tag.txt"
	  container('slave') {
	    withCredentials([usernamePassword(credentialsId: 'ecp-nexus-ecp-build', passwordVariable: 'repoPassword', usernameVariable: 'repoUser')]) {
	      sh """
                  REPO_PASSWORD='${repoPassword}' \
                  REPO_USERNAME='${repoUser}' \
                  GOOS=linux TAG='${publishVersion}' make prepare-release
                 """
	    }
	  }
	}
	// Tagging phase
	if (releaseBuild) {
	  stage('tagging') {
	    echo 'Tagging phase'
	    sshagent(['kublr-jenkins-ci']) {
	      sh """
                  git config user.email '${gitTaggerEmail}'
                  git config user.name  '${gitTaggerName}'

                  git tag -a 'v${publishVersion}' -m 'passed CI'

                  mkdir -p ~/.ssh
                  echo 'Host *' >> ~/.ssh/config
                  echo 'BatchMode=yes' >> ~/.ssh/config
                  echo 'StrictHostKeyChecking=no' >> ~/.ssh/config
  
                  git push origin 'v${publishVersion}'
                 """
	    }
	  }
	}
	stage('Save build info') { archiveArtifacts artifacts: 'tag.txt', onlyIfSuccessful: true }
      }
    } catch (Exception e) {
      currentBuild.result = 'Build Failed: ' + e
      buildStatus = "Failure: " + e
    } finally {
      stage('Slack notification') {
	String buildColor = currentBuild.result == null ? "#399E5A" : "#C03221"
	String buildEmoji = currentBuild.result == null ? ":ok_hand:" : ":thumbsdown:"
	String changes = summarizeBuild(currentBuild)
	slackMessage = "${buildEmoji} Build result for ${env.JOB_NAME} ${env.BRANCH_NAME} is: ${buildStatus}\n\n${changes}\n\nSee details at ${env.BUILD_URL}"
	slackSend color: buildColor, message: slackMessage
      }
    }    
  }
}
