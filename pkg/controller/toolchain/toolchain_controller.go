package toolchain

import (
	"context"
	"fmt"

	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_toolchain")

const ToolchainName = "codeready-toolchain"

// Add creates a new Toolchain Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	dynamicClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		log.Error(err, "Failed to add Toolchain controller to manager")
		return err
	}
	return add(mgr, newReconciler(mgr, dynamicClient))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, dynamicClient dynamic.Interface) *ReconcileToolchain {
	log.Info("Adding new Toolchain reconciler")
	return &ReconcileToolchain{
		client:            mgr.GetClient(),
		dynamicClient:     dynamicClient,
		dynamicRestMapper: mgr.GetRESTMapper(),
		scheme:            mgr.GetScheme(),
	}
}

// add adds a new controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileToolchain) error {
	// Create a new controller
	c, err := controller.New("toolchain-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Toolchain
	log.Info("configuring watcher on Toolchains")
	if err = c.Watch(&source.Kind{Type: &v1alpha1.Toolchain{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	log.Info("Toolchain reconciler successfully added")
	return nil
}

// blank assignment to verify that ReconcileTektonInstallation implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileToolchain{}

type ReconcileToolchain struct {
	client            client.Client
	dynamicClient     dynamic.Interface
	dynamicRestMapper meta.RESTMapper
	scheme            *runtime.Scheme
}

// Reconcile reads the state of the config for a Toolchain object and makes changes based on the state read
// and what is in the Toolchain.Spec
func (r *ReconcileToolchain) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Toolchain")

	// Fetch the Toolchain instance
	toolchain := &v1alpha1.Toolchain{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: ToolchainName}, toolchain); err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Toolchain not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// TODO make subscription namespace configurable
	subscriptionNs := "openshift-operators"
	reqLogger.Info(fmt.Sprintf("Creating operator subscription in %s", subscriptionNs))

	fmt.Printf("Toolchain obj: %v\n\n", *toolchain)
	if err := r.ensureComponents(reqLogger, toolchain, subscriptionNs); err != nil {
		reqLogger.Error(err, fmt.Sprintf("Failed to create operator subscription in %s", subscriptionNs))
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileToolchain) ensureComponents(logger logr.Logger, toolchain *v1alpha1.Toolchain, ns string) error {
	// TODO handle GroupVersionKind more dynamically
	// restMapping, err := r.dynamicRestMapper.RESTMapping(schema.GroupKind{Group: operators.GroupName, Kind: "Subscription"}, "v1alpha1")
	// if err != nil {
	// 	return err
	// }

	components, err := r.ProcessToList(toolchain)
	if err != nil {
		return err
	}
	// versionedToolchain, err := scheme.ConvertToVersion(toolchain, v1alpha1.GroupVersion)
	// if err != nil {
	// 	return err
	// }

	// unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(versionedToolchain)
	// if err != nil {
	// 	return err
	// }

	allErrors := []error{}
	fmt.Printf("Handling %v components\n", len(components.Items))
	for _, unstructuredObj := range components.Items {
		restMapping, mappingErr := r.dynamicRestMapper.RESTMapping(unstructuredObj.GroupVersionKind().GroupKind(), unstructuredObj.GroupVersionKind().Version)
		if mappingErr != nil {
			allErrors = append(allErrors, mappingErr)
			continue
		}

		fmt.Printf("Creating resource %v\n", unstructuredObj)
		_, err = r.dynamicClient.Resource(restMapping.Resource).Namespace(ns).Create(&unstructuredObj, metav1.CreateOptions{})
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}
	if len(allErrors) > 0 {
		return utilerrors.NewAggregate(allErrors)
	}

	return nil
}

func (r *ReconcileToolchain) ProcessToList(toolchain *v1alpha1.Toolchain) (*unstructured.UnstructuredList, error) {
	versionedToolchain, err := r.scheme.ConvertToVersion(toolchain, v1alpha1.GroupVersion)
	if err != nil {
		return nil, err
	}
	unstructuredComponent, err := runtime.DefaultUnstructuredConverter.ToUnstructured(versionedToolchain)
	if err != nil {
		return nil, err
	}

	return r.ProcessToListFromUnstructured(&unstructured.Unstructured{Object: unstructuredComponent})
}

func (r *ReconcileToolchain) ProcessToListFromUnstructured(unstructuredComponent *unstructured.Unstructured) (*unstructured.UnstructuredList, error) {

	toolchainSpec := unstructuredComponent.Object["spec"]
	toolchainSpecMap, ok := toolchainSpec.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to retrieve components, spec does not appear to have a valid structure")
	}
	// convert the template into something we iterate over as a list
	if err := unstructured.SetNestedField(unstructuredComponent.Object, toolchainSpecMap["components"], "items"); err != nil {
		return nil, err
	}
	return unstructuredComponent.ToList()
}
