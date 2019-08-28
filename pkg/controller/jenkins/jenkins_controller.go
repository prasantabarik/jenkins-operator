package jenkins

import (
	"context"
	"fmt"
	"jenkins-operator/pkg/service/jenkins"
	"jenkins-operator/pkg/service/platform"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	errorsf "github.com/pkg/errors"
	v2v1alpha1 "jenkins-operator/pkg/apis/v2/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	StatusInstall          = "installing"
	StatusFailed           = "failed"
	StatusCreated          = "created"
	StatusConfiguring      = "configuring"
	StatusConfigured       = "configured"
	StatusExposeStart      = "exposing config"
	StatusExposeFinish     = "config exposed"
	StatusIntegrationStart = "integration started"
	StatusReady            = "ready"
	DefaultRequeueTime     = 30
)

var log = logf.Log.WithName("controller_jenkins")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Jenkins Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	client := mgr.GetClient()
	platformService, _ := platform.NewPlatformService(scheme)

	jenkinsService := jenkins.NewJenkinsService(platformService, client)
	return &ReconcileJenkins{
		client:  client,
		scheme:  scheme,
		service: jenkinsService,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("jenkins-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*v2v1alpha1.Jenkins)
			newObject := e.ObjectNew.(*v2v1alpha1.Jenkins)
			if oldObject.Status != newObject.Status {
				return false
			}
			return true
		},
	}

	// Watch for changes to primary resource Jenkins
	err = c.Watch(&source.Kind{Type: &v2v1alpha1.Jenkins{}}, &handler.EnqueueRequestForObject{}, p)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileJenkins implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileJenkins{}

// ReconcileJenkins reconciles a Jenkins object
type ReconcileJenkins struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	service jenkins.JenkinsService
}

// Reconcile reads that state of the cluster for a Jenkins object and makes changes based on the state read
// and what is in the Jenkins.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJenkins) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling has been started")

	// Fetch the Jenkins instance
	instance := &v2v1alpha1.Jenkins{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.Status.Status == "" || instance.Status.Status == StatusFailed {
		reqLogger.Info("Installation has been started")
		err = r.updateStatus(instance, StatusInstall)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	instance, err = r.service.Install(*instance)
	if err != nil {
		r.updateStatus(instance, StatusFailed)
		return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, errorsf.Wrapf(err, "Installation has been failed")
	}

	if instance.Status.Status == StatusInstall {
		reqLogger.Info("Installation has finished")
		err = r.updateStatus(instance, StatusReady)
		if err != nil {
			return reconcile.Result{RequeueAfter: DefaultRequeueTime * time.Second}, err
		}
	}

	reqLogger.Info("Reconciling has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileJenkins) updateStatus(instance *v2v1alpha1.Jenkins, newStatus string) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	currentStatus := instance.Status.Status
	instance.Status.Status = newStatus
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		err := r.client.Update(context.TODO(), instance)
		if err != nil {
			return errorsf.Wrapf(err, "Couldn't update status from '%v' to '%v'", currentStatus, newStatus)
		}
	}
	reqLogger.Info(fmt.Sprintf("Status has been updated to '%v'", newStatus))
	return nil
}

func (r ReconcileJenkins) updateAvailableStatus(instance *v2v1alpha1.Jenkins, value bool) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			err := r.client.Update(context.TODO(), instance)
			if err != nil {
				return errorsf.Wrapf(err, "Couldn't update availability status to %v", value)
			}
		}
		reqLogger.Info(fmt.Sprintf("Availability status has been updated to '%v'", value))
	}
	return nil
}
