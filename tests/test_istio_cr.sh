#!/bin/bash

if [ "$1" == "" ]; then
    echo -e "\nUsage is:\n  test_istio_cr.sh create\n  test_istio_cr.sh delete\n"
    exit 1
fi

if [ "$1" == "create" ] ; then
    # install istio CR
    kubectl apply -f cr/ccp-istio-1.1.8-cr.yaml
    for i in $(seq 1 120) ; do
        kubectl get istio
        if kubectl get istio | grep IstioInstalledActive ; then
            echo -e "\nIstio CR reached active state!"
            kubectl get pods -n=istio-system
            break
        else
            if [ "$i" -eq "120" ] ; then
                echo -e "\nERROR: Istio CR did not reach active state and istio was not installed successfully, waited for 10 minutes.\n"
                kubectl get pods -n=istio-system
                echo "ccp-istio-operator pod's logs below:"
                kubectl get pods | grep ccp-istio-operator | awk '{print $1}' | xargs kubectl logs
                exit 1
            fi
            echo -e "\nWaiting for istio CR to be active ..."
            kubectl get pods -n=istio-system
            sleep 5
        fi
    done
elif [ "$1" == "delete" ] ; then
    # delete istio CR
    kubectl delete -f cr/ccp-istio-1.1.8-cr.yaml 
    for i in $(seq 1 120) ; do
        kubectl get pods -n=istio-system
        if [ $(kubectl get pods -n=istio-system | wc -l) -gt 1 ] ; then
            if [ "$i" -eq "120" ] ; then
                echo -e "\nERROR: Istio was not deleted successfully, waited for 10 minutes.\n"
                kubectl get istio
                kubectl get pods -n=istio-system
                echo "ccp-istio-operator pod's logs below:"
                kubectl get pods | grep ccp-istio-operator | awk '{print $1}' | xargs kubectl logs
                exit 1
            fi
            echo -e "\nWaiting for istio to be deleted ..."
            sleep 5
        else
            echo -e "\nIstio deleted successfully!"
            break
        fi
    done
fi
