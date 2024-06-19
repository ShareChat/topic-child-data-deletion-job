pipeline {
  agent {
    kubernetes {
      label 'sc-live-topic-child-database-cleanup-producer'
      yaml """
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: dind
    image: sc-mum-armory.platform.internal/devops/dind:v1
    securityContext:
      privileged: true
    volumeMounts:
      - name: dind-storage
        mountPath: /var/lib/docker
  - name: builder
    image: sc-mum-armory.platform.internal/devops/builder-image-golang-1.18-armory
    command:
    - sleep
    - infinity
    env:
    - name: DOCKER_HOST
      value: tcp://localhost:2375
    - name: DOCKER_BUILDKIT
      value: "0"
    volumeMounts:
      - name: jenkins-sa
        mountPath: /root/.gcp/
  volumes:
    - name: dind-storage
      emptyDir: {}
    - name: jenkins-sa
      secret:
        secretName: jenkins-sa
"""
    }
  }

  environment {
     sc_regions="mumbai"
     app="sc-live-topic-child-database-cleanup-producer"
     buildarg_DEPLOYMENT_ID="sc-live-topic-child-database-cleanup-producer-$GIT_COMMIT"
     buildarg_GITHUB_TOKEN=credentials('github-access')
  }

  stages {

    stage('build') {
      steps {
        container('builder') {
            sh 'armory build'
        }
      }
    }

    stage('push') {
      when {
        anyOf {
          branch 'main'
        }
      }
      steps {
        container('builder') {
          sh 'armory push'
        }
      }
    }
  }
}