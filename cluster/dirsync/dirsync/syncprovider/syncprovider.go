// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package syncprovider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gosimple/slug"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const directoryOwnedAnnotation = "octelium.com/directory-provider-owned"

const listPageSize = 300

const maxNameLen = 40

const userTagLen = 8

type User struct {
	ExternalID  string
	Email       string
	DisplayName string
	FirstName   string
	LastName    string
	Locale      string
	PicURL      string
	IsDisabled  bool
}

type Group struct {
	ExternalID        string
	DisplayName       string
	MemberExternalIDs []string
}

type Source interface {
	ListUsers(ctx context.Context) ([]*User, error)
	ListGroups(ctx context.Context) ([]*Group, error)
}

type Reconciler struct {
	octeliumC octeliumc.ClientInterface
	coreSrv   *admin.Server
	dp        *enterprisev1.DirectoryProvider
}

func NewReconciler(octeliumC octeliumc.ClientInterface, dp *enterprisev1.DirectoryProvider) *Reconciler {
	return &Reconciler{
		octeliumC: octeliumC,
		dp:        dp,
		coreSrv: admin.NewServer(&admin.Opts{
			OcteliumC:  octeliumC,
			IsEmbedded: true,
		}),
	}
}

func (r *Reconciler) genUserName(u *User) string {
	return r.genReadableName(u.Email, u.ExternalID)
}

func (r *Reconciler) genGroupName(g *Group) string {
	return r.genReadableName(g.DisplayName, g.ExternalID)
}

func (r *Reconciler) genReadableName(readable, externalID string) string {
	if readable == "" {
		return r.genName(externalID)
	}

	tag := externalIDTag(externalID)
	suffixBudget := maxNameLen - len(r.dp.Status.Id) - 1
	labelBudget := suffixBudget - len(tag) - 1
	if labelBudget < 0 {
		labelBudget = 0
	}

	label := slug.Make(readable)
	if len(label) > labelBudget {
		label = label[:labelBudget]
	}
	label = strings.Trim(label, "-")

	suffix := tag
	if label != "" {
		suffix = fmt.Sprintf("%s-%s", label, tag)
	}

	name := fmt.Sprintf("%s-%s", r.dp.Status.Id, suffix)
	if apivalidation.ValidateName(name, 0, 0) == nil {
		return name
	}

	return r.genName(externalID)
}

func (r *Reconciler) genName(externalID string) string {
	name := fmt.Sprintf("%s-%s", r.dp.Status.Id, slug.Make(externalID))
	if apivalidation.ValidateName(name, 0, 0) == nil {
		return name
	}

	sum := sha256.Sum256([]byte(externalID))
	h := hex.EncodeToString(sum[:])

	avail := maxNameLen - len(r.dp.Status.Id) - 1
	if avail > len(h) {
		avail = len(h)
	}

	return fmt.Sprintf("%s-%s", r.dp.Status.Id, h[:avail])
}

func externalIDTag(externalID string) string {
	s := strings.ReplaceAll(slug.Make(externalID), "-", "")
	if len(s) >= userTagLen {
		return s[:userTagLen]
	}

	sum := sha256.Sum256([]byte(externalID))
	return hex.EncodeToString(sum[:])[:userTagLen]
}

func (r *Reconciler) Sync(ctx context.Context, src Source) error {
	users, err := src.ListUsers(ctx)
	if err != nil {
		return err
	}

	groups, err := src.ListGroups(ctx)
	if err != nil {
		return err
	}

	desiredUserNames := make(map[string]struct{}, len(users))
	for _, u := range users {
		if u == nil || u.ExternalID == "" {
			continue
		}
		desiredUserNames[r.genUserName(u)] = struct{}{}
	}

	desiredGroupNames := make(map[string]struct{}, len(groups))
	for _, g := range groups {
		if g == nil || g.ExternalID == "" {
			continue
		}
		desiredGroupNames[r.genGroupName(g)] = struct{}{}
	}

	userNameByExternalID := make(map[string]string, len(users))
	for _, u := range users {
		if u == nil || u.ExternalID == "" {
			continue
		}
		coreUsr, err := r.upsertUser(ctx, u)
		if err != nil {
			zap.L().Warn("Could not upsert User during directory sync",
				zap.String("externalID", u.ExternalID), zap.Error(err))
			continue
		}
		userNameByExternalID[u.ExternalID] = coreUsr.Metadata.Name
	}

	managedGroupNames, err := r.listManagedGroupNames(ctx)
	if err != nil {
		return err
	}

	desiredGroupsByUser := make(map[string]map[string]struct{})
	for _, g := range groups {
		if g == nil || g.ExternalID == "" {
			continue
		}
		coreGrp, err := r.upsertGroup(ctx, g)
		if err != nil {
			zap.L().Warn("Could not upsert Group during directory sync",
				zap.String("externalID", g.ExternalID), zap.Error(err))
			continue
		}
		grpName := coreGrp.Metadata.Name
		managedGroupNames[grpName] = struct{}{}

		for _, memberExternalID := range g.MemberExternalIDs {
			usrName, ok := userNameByExternalID[memberExternalID]
			if !ok {
				continue
			}
			if desiredGroupsByUser[usrName] == nil {
				desiredGroupsByUser[usrName] = map[string]struct{}{}
			}
			desiredGroupsByUser[usrName][grpName] = struct{}{}
		}
	}

	for _, usrName := range userNameByExternalID {
		if err := r.reconcileUserGroups(ctx, usrName, managedGroupNames, desiredGroupsByUser[usrName]); err != nil {
			zap.L().Warn("Could not reconcile User groups during directory sync",
				zap.String("user", usrName), zap.Error(err))
		}
	}

	if err := r.pruneUsers(ctx, desiredUserNames); err != nil {
		zap.L().Warn("Could not prune Users during directory sync", zap.Error(err))
	}

	if err := r.pruneGroups(ctx, desiredGroupNames); err != nil {
		zap.L().Warn("Could not prune Groups during directory sync", zap.Error(err))
	}

	return nil
}

func (r *Reconciler) upsertUser(ctx context.Context, u *User) (*corev1.User, error) {
	name := r.genUserName(u)

	dpUsr, err := r.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{Name: name})
	if err == nil {
		coreUsr, err := r.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{Uid: dpUsr.Status.UserRef.Uid})
		if err != nil {
			return nil, err
		}
		r.applyUserSpec(coreUsr, u)
		return r.coreSrv.UpdateUser(ctx, coreUsr)
	}
	if !grpcerr.IsNotFound(err) {
		return nil, err
	}

	var coreUsr *corev1.User
	directoryOwned := false
	if u.Email != "" {
		usrList, err := r.octeliumC.CoreC().ListUser(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("spec.email", u.Email),
			},
		})
		if err != nil {
			return nil, err
		}
		if len(usrList.Items) > 0 {
			coreUsr = usrList.Items[0]
			directoryOwned = r.isDirectoryOwnedUser(ctx, coreUsr.Metadata.Uid)
		}
	}

	if coreUsr != nil {
		r.applyUserSpec(coreUsr, u)
		coreUsr, err = r.coreSrv.UpdateUser(ctx, coreUsr)
		if err != nil {
			return nil, err
		}
	} else {
		newUsr := &corev1.User{
			Metadata: &metav1.Metadata{Name: name},
			Spec:     &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
			Status:   &corev1.User_Status{},
		}
		r.applyUserSpec(newUsr, u)
		coreUsr, err = r.coreSrv.CreateUser(ctx, newUsr)
		if err != nil {
			return nil, err
		}
		directoryOwned = true
	}

	metadata := &metav1.Metadata{Name: name}
	if directoryOwned {
		metadata.Annotations = map[string]string{directoryOwnedAnnotation: "true"}
	}
	if _, err := r.octeliumC.EnterpriseC().CreateDirectoryProviderUser(ctx, &enterprisev1.DirectoryProviderUser{
		Metadata: metadata,
		Spec:     &enterprisev1.DirectoryProviderUser_Spec{},
		Status: &enterprisev1.DirectoryProviderUser_Status{
			UserRef:              umetav1.GetObjectReference(coreUsr),
			DirectoryProviderRef: umetav1.GetObjectReference(r.dp),
		},
	}); err != nil {
		return nil, err
	}

	return coreUsr, nil
}

func (r *Reconciler) isDirectoryOwnedUser(ctx context.Context, uid string) bool {
	links, err := r.octeliumC.EnterpriseC().ListDirectoryProviderUser(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.userRef.uid", uid),
		},
	})
	if err != nil {
		return false
	}

	for _, link := range links.Items {
		if link.Metadata != nil &&
			link.Metadata.Annotations[directoryOwnedAnnotation] == "true" {
			return true
		}
	}

	return false
}

func (r *Reconciler) applyUserSpec(usr *corev1.User, u *User) {
	if usr.Metadata == nil {
		usr.Metadata = &metav1.Metadata{}
	}
	usr.Metadata.DisplayName = u.DisplayName
	usr.Metadata.PicURL = u.PicURL

	if usr.Spec == nil {
		usr.Spec = &corev1.User_Spec{}
	}
	usr.Spec.Type = corev1.User_Spec_HUMAN
	usr.Spec.Email = u.Email
	usr.Spec.IsDisabled = u.IsDisabled
	usr.Spec.Info = &corev1.User_Spec_Info{
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Locale:    u.Locale,
	}
}

func (r *Reconciler) upsertGroup(ctx context.Context, g *Group) (*corev1.Group, error) {
	name := r.genGroupName(g)

	dpGrp, err := r.octeliumC.EnterpriseC().GetDirectoryProviderGroup(ctx, &rmetav1.GetOptions{Name: name})
	if err == nil {
		coreGrp, err := r.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{Uid: dpGrp.Status.GroupRef.Uid})
		if err != nil {
			return nil, err
		}
		coreGrp.Metadata.DisplayName = g.DisplayName
		return r.coreSrv.UpdateGroup(ctx, coreGrp)
	}
	if !grpcerr.IsNotFound(err) {
		return nil, err
	}

	coreGrp, err := r.coreSrv.CreateGroup(ctx, &corev1.Group{
		Metadata: &metav1.Metadata{
			Name:        name,
			DisplayName: g.DisplayName,
		},
		Spec:   &corev1.Group_Spec{},
		Status: &corev1.Group_Status{},
	})
	if err != nil {
		return nil, err
	}

	if _, err := r.octeliumC.EnterpriseC().CreateDirectoryProviderGroup(ctx, &enterprisev1.DirectoryProviderGroup{
		Metadata: &metav1.Metadata{Name: name},
		Spec:     &enterprisev1.DirectoryProviderGroup_Spec{},
		Status: &enterprisev1.DirectoryProviderGroup_Status{
			GroupRef:             umetav1.GetObjectReference(coreGrp),
			DirectoryProviderRef: umetav1.GetObjectReference(r.dp),
		},
	}); err != nil {
		return nil, err
	}

	return coreGrp, nil
}

func (r *Reconciler) reconcileUserGroups(ctx context.Context,
	usrName string, managed, desired map[string]struct{}) error {
	usr, err := r.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{Name: usrName})
	if err != nil {
		return err
	}

	final := make([]string, 0, len(usr.Spec.Groups))
	seen := map[string]struct{}{}

	for _, gName := range usr.Spec.Groups {
		if _, isManaged := managed[gName]; isManaged {
			continue
		}
		if _, ok := seen[gName]; ok {
			continue
		}
		seen[gName] = struct{}{}
		final = append(final, gName)
	}

	for gName := range desired {
		if _, ok := seen[gName]; ok {
			continue
		}
		seen[gName] = struct{}{}
		final = append(final, gName)
	}

	if equalStringSet(usr.Spec.Groups, final) {
		return nil
	}

	usr.Spec.Groups = final
	_, err = r.coreSrv.UpdateUser(ctx, usr)
	return err
}

func (r *Reconciler) listManagedGroupNames(ctx context.Context) (map[string]struct{}, error) {
	out := map[string]struct{}{}

	all, err := r.listAllDPGroups(ctx)
	if err != nil {
		return nil, err
	}

	for _, itm := range all {
		if itm.Status == nil || itm.Status.GroupRef == nil {
			continue
		}
		name := itm.Status.GroupRef.Name
		if name == "" {
			g, err := r.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{Uid: itm.Status.GroupRef.Uid})
			if err != nil {
				continue
			}
			name = g.Metadata.Name
		}
		out[name] = struct{}{}
	}

	return out, nil
}

func (r *Reconciler) pruneUsers(ctx context.Context, desired map[string]struct{}) error {
	all, err := r.listAllDPUsers(ctx)
	if err != nil {
		return err
	}

	for _, dpUsr := range all {
		if _, ok := desired[dpUsr.Metadata.Name]; ok {
			continue
		}
		r.deleteUser(ctx, dpUsr)
	}

	return nil
}

func (r *Reconciler) pruneGroups(ctx context.Context, desired map[string]struct{}) error {
	all, err := r.listAllDPGroups(ctx)
	if err != nil {
		return err
	}

	for _, dpGrp := range all {
		if _, ok := desired[dpGrp.Metadata.Name]; ok {
			continue
		}
		r.deleteGroup(ctx, dpGrp)
	}

	return nil
}

func (r *Reconciler) deleteUser(ctx context.Context, dpUsr *enterprisev1.DirectoryProviderUser) {
	owned := dpUsr.Metadata != nil &&
		dpUsr.Metadata.Annotations[directoryOwnedAnnotation] == "true"
	if !owned && dpUsr.Status != nil && dpUsr.Status.UserRef != nil {
		if usr, err := r.octeliumC.CoreC().GetUser(ctx,
			&rmetav1.GetOptions{Uid: dpUsr.Status.UserRef.Uid}); err == nil {
			owned = usr.Metadata.Name == dpUsr.Metadata.Name
		}
	}

	if _, err := r.octeliumC.EnterpriseC().DeleteDirectoryProviderUser(ctx, &rmetav1.DeleteOptions{
		Uid: dpUsr.Metadata.Uid,
	}); err != nil {
		if !grpcerr.IsNotFound(err) {
			zap.L().Warn("Could not delete DirectoryProviderUser",
				zap.String("name", dpUsr.Metadata.Name), zap.Error(err))
			return
		}
	}

	if dpUsr.Status == nil || dpUsr.Status.UserRef == nil {
		return
	}

	others, err := r.octeliumC.EnterpriseC().ListDirectoryProviderUser(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.userRef.uid", dpUsr.Status.UserRef.Uid),
		},
	})
	if err == nil && len(others.Items) > 0 {
		return
	}
	if !owned {
		return
	}

	if _, err := r.octeliumC.CoreC().DeleteUser(ctx, &rmetav1.DeleteOptions{
		Uid: dpUsr.Status.UserRef.Uid,
	}); err != nil {
		if !grpcerr.IsNotFound(err) {
			zap.L().Warn("Could not delete User",
				zap.String("dpUser", dpUsr.Metadata.Name), zap.Error(err))
		}
	}
}

func (r *Reconciler) deleteGroup(ctx context.Context, dpGrp *enterprisev1.DirectoryProviderGroup) {
	if _, err := r.octeliumC.EnterpriseC().DeleteDirectoryProviderGroup(ctx, &rmetav1.DeleteOptions{
		Uid: dpGrp.Metadata.Uid,
	}); err != nil {
		if !grpcerr.IsNotFound(err) {
			zap.L().Warn("Could not delete DirectoryProviderGroup",
				zap.String("name", dpGrp.Metadata.Name), zap.Error(err))
			return
		}
	}

	if dpGrp.Status == nil || dpGrp.Status.GroupRef == nil {
		return
	}

	others, err := r.octeliumC.EnterpriseC().ListDirectoryProviderGroup(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.groupRef.uid", dpGrp.Status.GroupRef.Uid),
		},
	})
	if err == nil && len(others.Items) > 0 {
		return
	}

	if _, err := r.octeliumC.CoreC().DeleteGroup(ctx, &rmetav1.DeleteOptions{
		Uid: dpGrp.Status.GroupRef.Uid,
	}); err != nil {
		if !grpcerr.IsNotFound(err) {
			zap.L().Warn("Could not delete Group",
				zap.String("dpGroup", dpGrp.Metadata.Name), zap.Error(err))
		}
	}
}

func (r *Reconciler) listAllDPUsers(ctx context.Context) ([]*enterprisev1.DirectoryProviderUser, error) {
	var out []*enterprisev1.DirectoryProviderUser
	var page uint32
	for {
		l, err := r.octeliumC.EnterpriseC().ListDirectoryProviderUser(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("status.directoryProviderRef.uid", r.dp.Metadata.Uid),
			},
			Paginate:     true,
			Page:         page,
			ItemsPerPage: listPageSize,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, l.Items...)
		if len(l.Items) < listPageSize {
			break
		}
		page++
	}
	return out, nil
}

func (r *Reconciler) listAllDPGroups(ctx context.Context) ([]*enterprisev1.DirectoryProviderGroup, error) {
	var out []*enterprisev1.DirectoryProviderGroup
	var page uint32
	for {
		l, err := r.octeliumC.EnterpriseC().ListDirectoryProviderGroup(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("status.directoryProviderRef.uid", r.dp.Metadata.Uid),
			},
			Paginate:     true,
			Page:         page,
			ItemsPerPage: listPageSize,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, l.Items...)
		if len(l.Items) < listPageSize {
			break
		}
		page++
	}
	return out, nil
}

func GetSecretValue(ctx context.Context, octeliumC octeliumc.ClientInterface, name string) (string, error) {
	if name == "" {
		return "", errors.Errorf("Empty Secret reference")
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{Name: name})
	if err != nil {
		return "", err
	}

	return uenterprisev1.ToSecret(sec).GetValueStr(), nil
}

func equalStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int, len(a))
	for _, x := range a {
		m[x]++
	}
	for _, x := range b {
		m[x]--
		if m[x] < 0 {
			return false
		}
	}
	return true
}
