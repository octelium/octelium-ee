// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package support

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/cluster"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultPortalURL = "https://portal.octelium.com"
	devPortalURL     = "https://portal-test.octelium.com"
	envPortalURL     = "OCTELIUM_SUPPORT_PORTAL_URL"
	supportEmail     = "support@octelium.com"

	requestTimeout  = 20 * time.Second
	infoTimeout     = 10 * time.Second
	uploadTimeout   = 2 * time.Minute
	downloadTimeout = 5 * time.Minute

	infoTTL      = time.Minute
	infoErrorTTL = 15 * time.Second

	maxSubjectLen           = 200
	minSubjectLen           = 3
	maxBodyLen              = 16000
	minReasonLen            = 10
	maxReasonLen            = 2000
	maxAttachmentSize       = 3 << 20
	maxAttachmentsPerTicket = 100
	maxAttachmentsPerMsg    = 10
	maxAttachmentNameLen    = 200
	maxDownloadSize         = 64 << 20
	downloadChunkSize       = 256 << 10
	maxReporterNameLen      = 200
	defaultAutoCloseDays    = 7
)

var (
	rgxUUID           = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	rgxIdempotencyKey = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)
	rgxAttachmentExt  = regexp.MustCompile(`\.([A-Za-z0-9]{1,8})$`)

	defaultExtensions = []string{
		"txt", "log", "conf", "cfg", "ini", "md", "json", "har", "ndjson", "jsonl",
		"yaml", "yml", "toml", "xml", "csv", "tsv", "pcap", "pcapng", "gz", "zip",
		"pdf", "png", "jpg", "jpeg", "gif", "webp", "eml",
	}

	getLicenseCredentialFunc = cluster.GetLicenseCredential
)

type infoEntry struct {
	key       string
	desc      *portalDescription
	err       error
	expiresAt time.Time
}

type Server struct {
	enterprisev1.UnimplementedSupportServiceServer
	octeliumC octeliumc.ClientInterface
	portal    *portalClient

	mu              sync.Mutex
	installationKey string
	info            *infoEntry
}

func NewServer(octeliumC octeliumc.ClientInterface) *Server {
	return &Server{
		octeliumC: octeliumC,
		portal:    newPortalClient(getPortalURL()),
	}
}

func getPortalURL() string {
	fallback := defaultPortalURL
	if ldflags.IsDev() {
		fallback = devPortalURL
	}

	val := strings.TrimSpace(os.Getenv(envPortalURL))
	if val == "" {
		return fallback
	}

	u, err := url.Parse(val)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		(u.Path != "" && u.Path != "/") ||
		(u.Scheme != "https" && !(u.Scheme == "http" && ldflags.IsDev())) {
		zap.L().Warn("Ignoring the invalid support portal URL", zap.String("url", val))
		return fallback
	}

	return u.Scheme + "://" + u.Host
}

func checkEnabled() error {
	if !ldflags.IsDev() {
		return status.Error(codes.Unimplemented, "Support requests are not available yet")
	}

	return nil
}

func (s *Server) GetSupportInfo(ctx context.Context, req *enterprisev1.GetSupportInfoRequest) (*enterprisev1.SupportInfo, error) {
	if err := checkEnabled(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	ret := &enterprisev1.SupportInfo{
		PortalURL:    s.portal.baseURL,
		SupportEmail: supportEmail,
		Limits:       getDefaultLimits(),
	}

	lic, err := getLicenseCredentialFunc(ctx, s.octeliumC)
	if err != nil {
		unavailable := &cluster.LicenseUnavailableError{}
		if !errors.As(err, &unavailable) {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.State = enterprisev1.SupportInfo_NO_LICENSE
		ret.Message = unavailable.Reason
		ret.ClusterDomain = s.getClusterDomain(ctx)
		ret.Environment = s.getEnvironment(ctx, nil)
		return ret, nil
	}

	ret.ClusterDomain = lic.ClusterDomain
	ret.Environment = s.getEnvironment(ctx, lic.License)

	if req.SkipPortal {
		return ret, nil
	}

	cred, err := s.newCredential(ctx, lic)
	if err != nil {
		return nil, err
	}

	desc, err := s.describe(ctx, cred)
	if err != nil {
		ret.State, ret.Message = getSupportInfoState(err)
		return ret, nil
	}

	ret.Organization = &enterprisev1.SupportInfo_Organization{
		Uid:         desc.Organization.UID,
		DisplayName: desc.Organization.DisplayName,
	}
	ret.SupportLevel = toSupportLevel(desc.SupportLevel)
	ret.InstallationID = desc.InstallationID
	ret.Limits = getLimits(desc)

	if desc.Entitled {
		ret.State = enterprisev1.SupportInfo_AVAILABLE
	} else {
		ret.State = enterprisev1.SupportInfo_NOT_ENTITLED
		ret.Message = "The subscription of the Cluster's License does not include technical support"
	}

	return ret, nil
}

func (s *Server) ListSupportRequest(ctx context.Context, req *enterprisev1.ListSupportRequestOptions) (*enterprisev1.SupportRequestList, error) {
	if err := checkEnabled(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}
	if len(req.Cursor) > 2048 {
		return nil, grpcutils.InvalidArg("Invalid cursor")
	}

	body := map[string]string{
		"status": "all",
	}
	switch req.Filter {
	case enterprisev1.ListSupportRequestOptions_OPEN:
		body["status"] = "open"
	case enterprisev1.ListSupportRequestOptions_CLOSED:
		body["status"] = "closed"
	}
	if req.Cursor != "" {
		body["cursor"] = req.Cursor
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	cred, err := s.getCredential(ctx)
	if err != nil {
		return nil, err
	}

	out := &portalTicketList{}
	if err := s.portal.do(ctx, cred, &portalRequest{
		method: http.MethodPost,
		path:   pathTicketsList,
		body:   body,
	}, out); err != nil {
		return nil, toGRPCError(err)
	}

	ret := &enterprisev1.SupportRequestList{
		NextCursor: out.Cursor,
	}
	for i := range out.Tickets {
		ret.Items = append(ret.Items, s.toSupportRequest(&out.Tickets[i]))
	}

	return ret, nil
}

func (s *Server) GetSupportRequest(ctx context.Context, req *enterprisev1.GetSupportRequestRequest) (*enterprisev1.SupportRequest, error) {
	if err := checkEnabled(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}
	if err := checkID(req.Id); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	cred, err := s.getCredential(ctx)
	if err != nil {
		return nil, err
	}

	return s.getDetail(ctx, cred, req.Id)
}

func (s *Server) ListSupportRequestEvent(ctx context.Context, req *enterprisev1.ListSupportRequestEventOptions) (*enterprisev1.SupportRequestEventList, error) {
	if err := checkEnabled(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}
	if err := checkID(req.Id); err != nil {
		return nil, err
	}
	if req.Cursor == "" || len(req.Cursor) > 2048 {
		return nil, grpcutils.InvalidArg("Invalid cursor")
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	cred, err := s.getCredential(ctx)
	if err != nil {
		return nil, err
	}

	out := &portalEventList{}
	if err := s.portal.do(ctx, cred, &portalRequest{
		method: http.MethodPost,
		path:   getTicketPath(req.Id, "events", "list"),
		body: map[string]string{
			"cursor": req.Cursor,
		},
	}, out); err != nil {
		return nil, toGRPCError(err)
	}

	return &enterprisev1.SupportRequestEventList{
		Items:      toEvents(out.Events),
		NextCursor: out.Cursor,
	}, nil
}

func (s *Server) CreateSupportRequest(ctx context.Context, req *enterprisev1.CreateSupportRequestRequest) (*enterprisev1.SupportRequest, error) {
	if err := checkEnabled(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	subject, err := getSubject(req.Subject)
	if err != nil {
		return nil, err
	}

	description, err := getText(req.Description, "Description", 1, maxBodyLen)
	if err != nil {
		return nil, err
	}

	idempotencyKey, err := getIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if idempotencyKey == "" {
		idempotencyKey = uuid.NewString()
	}

	switch req.Priority {
	case enterprisev1.SupportRequest_PRIORITY_UNKNOWN, enterprisev1.SupportRequest_P1,
		enterprisev1.SupportRequest_P2, enterprisev1.SupportRequest_P3, enterprisev1.SupportRequest_P4:
	default:
		return nil, grpcutils.InvalidArg("Invalid priority")
	}

	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	cred, lic, err := s.getCredentialWithLicense(ctx)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"subject":     subject,
		"description": description,
		"priority":    fromPriority(req.Priority),
	}
	if environment := s.getEnvironment(ctx, lic.License); len(environment) > 0 {
		body["environment"] = environment
	}
	if reporter := getReporter(i); reporter != nil {
		body["reporter"] = reporter
	}

	out := &portalTicketResponse{}
	if err := s.portal.do(ctx, cred, &portalRequest{
		method:         http.MethodPost,
		path:           pathTickets,
		body:           body,
		idempotencyKey: idempotencyKey,
	}, out); err != nil {
		return nil, toGRPCError(err)
	}

	zap.L().Info("Created a support request",
		zap.Uint32("number", out.Ticket.Number),
		zap.String("priority", out.Ticket.Priority),
		zap.String("userUID", getUserUID(i)))

	return s.getDetailAfterUpdate(ctx, cred, out.Ticket.ID, &out.Ticket)
}

func (s *Server) AddSupportRequestMessage(ctx context.Context, req *enterprisev1.AddSupportRequestMessageRequest) (*enterprisev1.SupportRequest, error) {
	if err := checkEnabled(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}
	if err := checkID(req.Id); err != nil {
		return nil, err
	}

	text, err := getText(req.Body, "Message", 1, maxBodyLen)
	if err != nil {
		return nil, err
	}

	idempotencyKey, err := getIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	if len(req.AttachmentIDs) > maxAttachmentsPerMsg {
		return nil, grpcutils.InvalidArg("A message can reference at most %d files", maxAttachmentsPerMsg)
	}
	for _, attachmentID := range req.AttachmentIDs {
		if !rgxUUID.MatchString(attachmentID) {
			return nil, grpcutils.InvalidArg("Invalid file ID")
		}
	}

	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	cred, err := s.getCredential(ctx)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"body": text,
	}
	if len(req.AttachmentIDs) > 0 {
		body["attachmentIDs"] = req.AttachmentIDs
	}
	if reporter := getReporter(i); reporter != nil {
		body["reporter"] = reporter
	}

	out := &portalTicketResponse{}
	if err := s.portal.do(ctx, cred, &portalRequest{
		method:         http.MethodPost,
		path:           getTicketPath(req.Id, "messages"),
		body:           body,
		idempotencyKey: idempotencyKey,
	}, out); err != nil {
		return nil, toGRPCError(err)
	}

	zap.L().Info("Added a message to a support request",
		zap.String("id", req.Id),
		zap.Int("attachments", len(req.AttachmentIDs)),
		zap.String("userUID", getUserUID(i)))

	return s.getDetailAfterUpdate(ctx, cred, req.Id, &out.Ticket)
}

func (s *Server) AddSupportRequestAttachment(ctx context.Context, req *enterprisev1.AddSupportRequestAttachmentRequest) (*enterprisev1.SupportRequest_Attachment, error) {
	if err := checkEnabled(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}
	if err := checkID(req.Id); err != nil {
		return nil, err
	}

	name, err := getAttachmentName(req.Name)
	if err != nil {
		return nil, err
	}

	if len(req.Data) == 0 {
		return nil, grpcutils.InvalidArg("The file is empty")
	}
	if len(req.Data) > maxAttachmentSize {
		return nil, grpcutils.InvalidArg("Files are limited to %d MB", maxAttachmentSize>>20)
	}

	idempotencyKey, err := getIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"X-Filename":       url.PathEscape(name),
		"X-Content-SHA256": getSHA256(req.Data),
	}

	switch req.Purpose {
	case enterprisev1.SupportRequest_Attachment_DIAGNOSTIC:
		headers["X-Attachment-Purpose"] = "DIAGNOSTIC"
	case enterprisev1.SupportRequest_Attachment_PURPOSE_UNKNOWN, enterprisev1.SupportRequest_Attachment_FILE:
	default:
		return nil, grpcutils.InvalidArg("Invalid purpose")
	}

	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	cred, err := s.getCredential(ctx)
	if err != nil {
		return nil, err
	}

	out := &portalAttachmentResponse{}
	if err := s.portal.do(ctx, cred, &portalRequest{
		method:         http.MethodPost,
		path:           getTicketPath(req.Id, "attachments"),
		raw:            req.Data,
		headers:        headers,
		idempotencyKey: idempotencyKey,
	}, out); err != nil {
		return nil, toGRPCError(err)
	}

	zap.L().Info("Added a file to a support request",
		zap.String("id", req.Id),
		zap.String("purpose", headers["X-Attachment-Purpose"]),
		zap.Int("size", len(req.Data)),
		zap.String("userUID", getUserUID(i)))

	return toAttachment(&out.Attachment), nil
}

func (s *Server) ResolveSupportRequest(ctx context.Context, req *enterprisev1.ResolveSupportRequestRequest) (*enterprisev1.SupportRequest, error) {
	if err := checkEnabled(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}
	if err := checkID(req.Id); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	cred, err := s.getCredential(ctx)
	if err != nil {
		return nil, err
	}

	out := &portalTicketResponse{}
	if err := s.portal.do(ctx, cred, &portalRequest{
		method: http.MethodPost,
		path:   getTicketPath(req.Id, "resolve"),
	}, out); err != nil {
		return nil, toGRPCError(err)
	}

	return s.getDetailAfterUpdate(ctx, cred, req.Id, &out.Ticket)
}

func (s *Server) EscalateSupportRequest(ctx context.Context, req *enterprisev1.EscalateSupportRequestRequest) (*enterprisev1.SupportRequest, error) {
	if err := checkEnabled(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}
	if err := checkID(req.Id); err != nil {
		return nil, err
	}

	reason, err := getText(req.Reason, "Reason", minReasonLen, maxReasonLen)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	cred, err := s.getCredential(ctx)
	if err != nil {
		return nil, err
	}

	out := &portalTicketResponse{}
	if err := s.portal.do(ctx, cred, &portalRequest{
		method: http.MethodPost,
		path:   getTicketPath(req.Id, "escalate"),
		body: map[string]string{
			"reason": reason,
		},
	}, out); err != nil {
		return nil, toGRPCError(err)
	}

	return s.getDetailAfterUpdate(ctx, cred, req.Id, &out.Ticket)
}

func (s *Server) DownloadSupportRequestAttachment(req *enterprisev1.DownloadSupportRequestAttachmentRequest, stream enterprisev1.SupportService_DownloadSupportRequestAttachmentServer) error {
	if err := checkEnabled(); err != nil {
		return err
	}
	if req == nil {
		return grpcutils.InvalidArg("Nil request")
	}
	if err := checkID(req.Id); err != nil {
		return err
	}
	if !rgxUUID.MatchString(req.AttachmentID) {
		return grpcutils.InvalidArg("Invalid file ID")
	}

	ctx, cancel := context.WithTimeout(stream.Context(), downloadTimeout)
	defer cancel()

	cred, err := s.getCredential(ctx)
	if err != nil {
		return err
	}

	resp, err := s.portal.open(ctx, cred, &portalRequest{
		method: http.MethodGet,
		path:   getTicketPath(req.Id, "attachments", req.AttachmentID),
	})
	if err != nil {
		return toGRPCError(err)
	}
	defer resp.Body.Close()

	if resp.ContentLength > maxDownloadSize {
		return status.Error(codes.FailedPrecondition, "The file is too large to be downloaded from the Cluster")
	}

	header := &enterprisev1.DownloadSupportRequestAttachmentResponse_Header{
		Name:        getDownloadName(resp.Header.Get("Content-Disposition")),
		ContentType: "application/octet-stream",
	}
	if resp.ContentLength > 0 {
		header.Size = uint64(resp.ContentLength)
	}

	if err := stream.Send(&enterprisev1.DownloadSupportRequestAttachmentResponse{
		Type: &enterprisev1.DownloadSupportRequestAttachmentResponse_Header_{
			Header: header,
		},
	}); err != nil {
		return err
	}

	reader := io.LimitReader(resp.Body, maxDownloadSize+1)
	buf := make([]byte, downloadChunkSize)
	total := 0
	for {
		n, err := io.ReadFull(reader, buf)
		if n > 0 {
			total += n
			if total > maxDownloadSize {
				return status.Error(codes.FailedPrecondition, "The file is too large to be downloaded from the Cluster")
			}

			if err := stream.Send(&enterprisev1.DownloadSupportRequestAttachmentResponse{
				Type: &enterprisev1.DownloadSupportRequestAttachmentResponse_Chunk{
					Chunk: slices.Clone(buf[:n]),
				},
			}); err != nil {
				return err
			}
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			if resp.ContentLength >= 0 && int64(total) != resp.ContentLength {
				zap.L().Warn("The support attachment download ended prematurely",
					zap.String("id", req.Id),
					zap.Int64("expected", resp.ContentLength),
					zap.Int("received", total))
				return status.Error(codes.Unavailable, "The file download was interrupted. Please try again")
			}
			return nil
		}
		if err != nil {
			return toGRPCError(err)
		}
	}
}

func (s *Server) getDetail(ctx context.Context, cred *credential, id string) (*enterprisev1.SupportRequest, error) {
	out := &portalDetail{}
	if err := s.portal.do(ctx, cred, &portalRequest{
		method: http.MethodGet,
		path:   getTicketPath(id),
	}, out); err != nil {
		return nil, toGRPCError(err)
	}

	return s.toSupportRequestDetail(out), nil
}

func (s *Server) getDetailAfterUpdate(ctx context.Context, cred *credential, id string, updated *portalTicket) (*enterprisev1.SupportRequest, error) {
	ret, err := s.getDetail(ctx, cred, id)
	if err == nil {
		return ret, nil
	}
	if updated == nil || updated.ID == "" {
		return nil, err
	}

	zap.L().Warn("Could not get the support request after updating it", zap.String("id", id), zap.Error(err))
	return s.toSupportRequest(updated), nil
}

func (s *Server) describe(ctx context.Context, cred *credential) (*portalDescription, error) {
	key := cred.cacheKey()
	now := time.Now()

	s.mu.Lock()
	entry := s.info
	s.mu.Unlock()

	if entry != nil && entry.key == key && now.Before(entry.expiresAt) {
		return entry.desc, entry.err
	}

	ctx, cancel := context.WithTimeout(ctx, infoTimeout)
	defer cancel()

	desc := &portalDescription{}
	err := s.portal.do(ctx, cred, &portalRequest{
		method: http.MethodGet,
		path:   pathSupport,
	}, desc)

	entry = &infoEntry{
		key:       key,
		expiresAt: now.Add(infoTTL),
	}
	if err != nil {
		zap.L().Debug("Could not describe the support of the Cluster", zap.Error(err))
		desc = nil
		entry.err = err
		entry.expiresAt = now.Add(infoErrorTTL)
	} else {
		entry.desc = desc
	}

	if !errors.Is(err, context.Canceled) {
		s.mu.Lock()
		s.info = entry
		s.mu.Unlock()
	}

	return desc, err
}

func (s *Server) getCredential(ctx context.Context) (*credential, error) {
	ret, _, err := s.getCredentialWithLicense(ctx)
	return ret, err
}

func (s *Server) getCredentialWithLicense(ctx context.Context) (*credential, *cluster.LicenseCredential, error) {
	lic, err := getLicenseCredentialFunc(ctx, s.octeliumC)
	if err != nil {
		unavailable := &cluster.LicenseUnavailableError{}
		if errors.As(err, &unavailable) {
			return nil, nil, status.Error(codes.FailedPrecondition, unavailable.Reason)
		}
		return nil, nil, grpcutils.InternalWithErr(err)
	}

	cred, err := s.newCredential(ctx, lic)
	if err != nil {
		return nil, nil, err
	}

	return cred, lic, nil
}

func (s *Server) newCredential(ctx context.Context, lic *cluster.LicenseCredential) (*credential, error) {
	if lic.ClusterDomain == "" {
		return nil, status.Error(codes.FailedPrecondition, "The Cluster domain is not set")
	}

	key, err := s.getInstallationKey(ctx)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return &credential{
		license:         lic.JWT,
		clusterDomain:   lic.ClusterDomain,
		installationKey: key,
	}, nil
}

func (s *Server) getClusterDomain(ctx context.Context) string {
	cc, err := s.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil || cc.Status == nil {
		return ""
	}

	return cc.Status.Domain
}

func getDefaultLimits() *enterprisev1.SupportInfo_Limits {
	return &enterprisev1.SupportInfo_Limits{
		MaxSubjectLength:         maxSubjectLen,
		MaxBodyLength:            maxBodyLen,
		MaxAttachmentSize:        maxAttachmentSize,
		MaxAttachmentsPerRequest: maxAttachmentsPerTicket,
		MaxAttachmentsPerMessage: maxAttachmentsPerMsg,
		AttachmentExtensions:     defaultExtensions,
		AutoCloseDays:            defaultAutoCloseDays,
	}
}

func getLimits(desc *portalDescription) *enterprisev1.SupportInfo_Limits {
	ret := getDefaultLimits()

	if desc.MaxBodyLength > 0 && desc.MaxBodyLength < ret.MaxBodyLength {
		ret.MaxBodyLength = desc.MaxBodyLength
	}
	if desc.Attachments.MaxBytes > 0 && desc.Attachments.MaxBytes < ret.MaxAttachmentSize {
		ret.MaxAttachmentSize = desc.Attachments.MaxBytes
	}
	if desc.Attachments.MaxPerTicket > 0 {
		ret.MaxAttachmentsPerRequest = desc.Attachments.MaxPerTicket
	}
	if len(desc.Attachments.Extensions) > 0 {
		ret.AttachmentExtensions = desc.Attachments.Extensions
	}
	if desc.AutoCloseDays > 0 {
		ret.AutoCloseDays = desc.AutoCloseDays
	}

	return ret
}

func getSupportInfoState(err error) (enterprisev1.SupportInfo_State, string) {
	perr := &portalError{}
	if errors.As(err, &perr) {
		switch perr.status {
		case http.StatusUnauthorized, http.StatusForbidden:
			return enterprisev1.SupportInfo_UNAUTHORIZED,
				getMessage(perr.message, "Octelium could not verify the License of the Cluster")
		case http.StatusNotFound:
			return enterprisev1.SupportInfo_DISABLED,
				"Octelium does not currently accept support requests from Clusters"
		case http.StatusTooManyRequests:
			return enterprisev1.SupportInfo_UNREACHABLE,
				"Too many requests were sent to the customer portal. Please try again later"
		default:
			return enterprisev1.SupportInfo_UNREACHABLE,
				"The customer portal is temporarily unavailable"
		}
	}

	return enterprisev1.SupportInfo_UNREACHABLE, "Could not reach the Octelium customer portal from the Cluster"
}

func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	if _, ok := status.FromError(err); ok {
		return err
	}

	perr := &portalError{}
	if errors.As(err, &perr) {
		switch perr.status {
		case http.StatusBadRequest, http.StatusLengthRequired,
			http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType:
			return status.Error(codes.InvalidArgument,
				getMessage(perr.message, "The customer portal rejected the request"))
		case http.StatusUnauthorized:
			return status.Error(codes.FailedPrecondition,
				getMessage(perr.message, "Octelium could not verify the License of the Cluster"))
		case http.StatusForbidden, http.StatusConflict, http.StatusGone:
			return status.Error(codes.FailedPrecondition,
				getMessage(perr.message, "The customer portal rejected the request"))
		case http.StatusNotFound:
			return status.Error(codes.NotFound, "The support request could not be found")
		case http.StatusTooManyRequests:
			return status.Error(codes.ResourceExhausted,
				getMessage(perr.message, "Too many requests. Please try again later"))
		default:
			return status.Error(codes.Unavailable, "The customer portal is temporarily unavailable")
		}
	}

	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "The request was canceled")
	}

	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr) {
		return status.Error(codes.Unavailable, "Could not reach the Octelium customer portal from the Cluster")
	}

	zap.L().Warn("Unexpected support error", zap.Error(err))
	return status.Error(codes.Unavailable, "The customer portal returned an invalid response")
}

func getMessage(msg string, fallback string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" || utf8.RuneCountInString(msg) > 500 {
		return fallback
	}

	r, size := utf8.DecodeRuneInString(msg)
	return string(unicode.ToUpper(r)) + msg[size:]
}

func checkID(id string) error {
	if !rgxUUID.MatchString(id) {
		return grpcutils.InvalidArg("Invalid support request ID")
	}

	return nil
}

func getTicketPath(id string, segments ...string) string {
	ret := pathTickets + "/" + url.PathEscape(strings.ToLower(id))
	for _, segment := range segments {
		ret += "/" + url.PathEscape(strings.ToLower(segment))
	}
	return ret
}

func cleanText(arg string) string {
	arg = strings.ReplaceAll(arg, "\r\n", "\n")
	arg = strings.ReplaceAll(arg, "\r", "\n")
	arg = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, arg)
	return strings.TrimSpace(arg)
}

func getText(arg string, name string, minLen, maxLen int) (string, error) {
	if !utf8.ValidString(arg) {
		return "", grpcutils.InvalidArg("%s is not valid UTF-8", name)
	}

	ret := cleanText(arg)
	l := utf8.RuneCountInString(ret)
	if l < minLen {
		if minLen <= 1 {
			return "", grpcutils.InvalidArg("%s is required", name)
		}
		return "", grpcutils.InvalidArg("%s must be at least %d characters", name, minLen)
	}
	if l > maxLen {
		return "", grpcutils.InvalidArg("%s must be at most %d characters", name, maxLen)
	}

	return ret, nil
}

func getSubject(arg string) (string, error) {
	ret, err := getText(arg, "Subject", minSubjectLen, maxSubjectLen)
	if err != nil {
		return "", err
	}
	if strings.ContainsAny(ret, "\n\t") {
		return "", grpcutils.InvalidArg("Subject must be a single line")
	}

	return ret, nil
}

func getIdempotencyKey(arg string) (string, error) {
	if arg == "" {
		return "", nil
	}
	if !rgxIdempotencyKey.MatchString(arg) {
		return "", grpcutils.InvalidArg("Invalid idempotency key")
	}

	return arg, nil
}

func getAttachmentName(arg string) (string, error) {
	if !utf8.ValidString(arg) {
		return "", grpcutils.InvalidArg("Invalid file name")
	}

	ret := strings.ReplaceAll(arg, "\\", "/")
	ret = path.Base(strings.TrimSpace(ret))
	ret = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`"<>|:*?/`, r) {
			return -1
		}
		return r
	}, ret)
	ret = strings.TrimSpace(ret)

	if ret == "" || ret == "." || len(ret) > maxAttachmentNameLen {
		return "", grpcutils.InvalidArg("Invalid file name")
	}

	match := rgxAttachmentExt.FindStringSubmatch(ret)
	if len(match) != 2 || !slices.Contains(defaultExtensions, strings.ToLower(match[1])) {
		return "", grpcutils.InvalidArg("This file type is not accepted")
	}

	return ret, nil
}

func getDownloadName(disposition string) string {
	if _, params, err := mime.ParseMediaType(disposition); err == nil {
		if name, err := getAttachmentName(params["filename"]); err == nil {
			return name
		}
	}

	return "attachment"
}

func getReporter(i *userctx.UserCtx) map[string]string {
	if i == nil || i.User == nil {
		return nil
	}

	ret := make(map[string]string)

	name := ""
	if i.User.Metadata != nil {
		name = strings.TrimSpace(i.User.Metadata.DisplayName)
		if name == "" {
			name = strings.TrimSpace(i.User.Metadata.Name)
		}
	}
	if name = cleanText(name); name != "" && utf8.RuneCountInString(name) <= maxReporterNameLen {
		ret["name"] = name
	}

	if i.User.Spec != nil {
		email := strings.ToLower(strings.TrimSpace(i.User.Spec.Email))
		if addr, err := mail.ParseAddress(email); err == nil && addr.Name == "" &&
			addr.Address == email && len(email) <= 254 {
			ret["email"] = email
		}
	}

	if len(ret) == 0 {
		return nil
	}

	return ret
}

func getUserUID(i *userctx.UserCtx) string {
	if i == nil || i.User == nil || i.User.Metadata == nil {
		return ""
	}

	return i.User.Metadata.Uid
}

func getSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
