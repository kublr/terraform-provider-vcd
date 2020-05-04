#!/usr/bin/env groovy

@NonCPS
static def summarizeBuild(b) {
    b.changeSets.collect { cs ->
        /Changes: / + cs.collect { entry ->
            / — ${entry.msg} by ${entry.author.fullName} /
        }.join('\n')
    }.join('\n')
}

repositories = [
        prod: [
                branch        : ['master', 'release/.*'],
	        goRepoUrl     : "https://repo.kublr.com/repository/gobinaries",
                credentialsId : "jenkins-nexus-beta"
        ],
        any : [
                branch        : ['.*'],
	        goRepoUrl     : "https://nexus.ecp.eastbanctech.com/repository/gobinaries",
                credentialsId : "ecp-nexus-ecp-build"
        ]
]


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
                        image: 'nexus.build.svc.cluster.local:5000/jenkinsci/jnlp-slave-ext:0.1.27')
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
	  publishVersion = releaseBuild ? "${srcVersion}-${BUILD_NUMBER}" : "${srcVersion}-${branchQual}.${BUILD_NUMBER}"
	  sh "echo ${publishVersion} > tag.txt"
	  container('slave') {
	    def repos = repositoryMatch()
	    repos.each { repoName, repo ->
	      println "publish in repository ${repoName}"
	      withCredentials([usernamePassword(credentialsId: repo.credentialsId, passwordVariable: 'repoPassword', usernameVariable: 'repoUser'), ])  {
		sh """
                  export REPO_PASSWORD='${repoPassword}'
                  export REPO_USERNAME='${repoUser}'
                  export GOBINARIES_REPO_URL='${repo.goRepoUrl}'
                  GOOS=linux make test
                  GOOS=linux   TAG='${publishVersion}' make prepare-release
                  GOOS=windows TAG='${publishVersion}' make prepare-release
                  GOOS=darwin  TAG='${publishVersion}' make prepare-release
                 """
	      }
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

String getBranchName() {
    String gitBranch = sh returnStdout: true, script: 'git rev-parse --abbrev-ref HEAD'
    return gitBranch.trim()
}
// Find all the repositories in which the pattern matches the branch
def repositoryMatch() {
  def result = [:]
  String branchName = getBranchName()
  println "branchName ${branchName}"

  repositories.any { repoName, repo ->
    def match = false
    repo.branch.any { branch ->
      if (branchName.matches(branch)) {
	match = true
	return false // break closure
      }
    }

    if (match) {
      ['goRepoUrl', 'credentialsId'].each { field ->
	if (!repo.containsKey(field) || repo[field] == null) {
	  error "'${field}' field must be set for repository ${repoName}"
	}
      }
      result.put(repoName, repo)
    }
  }
  return result
}
