#!groovy

job = env.JOB_BASE_NAME.toString()
if (job == "recheck") {
  job = "premerge"
}
if (job == "onmerge") {
  job = "postmerge"
}
if (job == "ontag") {
  job = "release"
}

pipeline {
  options {
    timeout(time: 30, unit: 'MINUTES')
  }
  agent {
    node {
      label 'ci-runner'
      customWorkspace "/home/jenkins/go/src/wwwin-github.cisco.com/CPSG/ccp-istio-operator"
    }
  }
  // Do not set a timeout at this level.  The timeout blocks below have been designed to give the VM cleanup sufficient
  // time to run if any/all other timeout blocks are hit.  Unfortunately it seems that we cannot set a top level
  // timeout for multiple stages, so we need to set a timeout for each stage independently.  In order to set a
  // collective timeout for everything outside of the post block, we'd have to do everything in one stage.
  environment {
    BUILD_TYPE = "${job}"
  }
  stages {
    stage('Run Tests') {
      options {
        timeout(time: 10, unit: 'MINUTES')
      }
      steps {
        sh '''
        make test
        '''
      }
    }
    stage('Docker build and upload (premerge)') {
      options {
        timeout(time: 30, unit: 'MINUTES')
      }
      steps {
        withCredentials([[$class: "UsernamePasswordMultiBinding",
                          credentialsId: "CPSG-ccp-istio-operator-registry",
                          usernameVariable: "USERNAME",
                          passwordVariable: "PASSWORD"]]) {
          sh '''
            docker login -u "$USERNAME" -p "$PASSWORD" registry-write.ci.ciscolabs.com
            export REGISTRY="registry-write.ci.ciscolabs.com/$USERNAME"
            make docker-build
            make docker-push
            docker logout registry-write.ci.ciscolabs.com
          '''
        }
      }
    }
    stage('Helm chart build and upload (premerge)') {
      options {
        timeout(time: 30, unit: 'MINUTES')
      }
      steps {
        withCredentials([[$class: "UsernamePasswordMultiBinding",
                          credentialsId: "CPSG-ccp-istio-operator-repo",
                          usernameVariable: "HELM_REPO_USERNAME",
                          passwordVariable: "HELM_REPO_PASSWORD"]]) {
          sh '''
            make helm-package
            make helm-upload
            '''
        }
      }
    }
  }
  post {
      success {
        slackSend(
          color: "#00FF00",
          message: "👍 SUCCESS ${env.JOB_NAME} ${env.BUILD_URL}",
          channel: "${env.ghprbPullAuthorLoginMention}"
        )
      }
      failure {
        slackSend(
          color: "#FF0000",
          message: "👎 FAIL ${env.JOB_NAME} \n${env.BUILD_URL} \n${env.BUILD_URL}console",
          channel: "${env.ghprbPullAuthorLoginMention}"
        )
      }
  }
}
