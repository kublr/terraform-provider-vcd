#!/usr/bin/env groovy

@NonCPS
static def summarizeBuild(b) {
    b.changeSets.collect { cs ->
        /Changes: / + cs.collect { entry ->
            / — ${entry.msg} by ${entry.author.fullName} /
        }.join('\n')
    }.join('\n')
}

podTemplate(
  label: 'kublrslave',
  containers: [
    containerTemplate(name: 'jnlp',  image: 'jenkinsci/jnlp-slave', args: '${computer.jnlpmac} ${computer.name}'),
    containerTemplate(name: 'slave', ttyEnabled: true, command: 'cat', args: '-v', image: 'nexus.build.svc.cluster.local:5000/jenkinsci/jnlp-slave-ext:0.1.15')
  ]) {
  node('kublrslave') {
    String buildStatus = 'Success'
    try {
      def APP_DIR = "${HOME}/go/src/github.com/kublr/govcloudair"
      stage('build'){
	sh "mkdir -p ${APP_DIR}"
	dir("${APP_DIR}") {	  
	  checkout scm
	  container('slave'){	  
	    sh "GOOS=linux GOPATH=${HOME}/go make test"
	  }
	}
      }
    } catch (Exception e) {
      currentBuild.result = 'Build Failed: ' + e
      buildStatus = "Failure: " + e
    } finally {
      stage('Slack notification') {
	String buildColor = currentBuild.result == null ? "#399E5A" : "#C03221"
	String buildEmoji = currentBuild.result == null ? ":ok_hand:" : ":thumbsdown:"
	String changes = summarizeBuild(currentBuild)
	// Message body
	slackMessage = "${buildEmoji} Build result for ${env.JOB_NAME} ${env.BRANCH_NAME} is: ${buildStatus}\n\n${changes}\n\nSee details at ${env.BUILD_URL}"
	slackSend color: buildColor, message: slackMessage
      }
    }
  }
}
