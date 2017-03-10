# SPEC file

%global c_vendor    %{_vendor}
%global gh_owner    %{_owner}
%global gh_cvspath  %{_cvspath}
%global gh_project  %{_project}

Name:      %{_package}
Version:   %{_version}
Release:   %{_release}%{?dist}
Summary:   ~#SHORTDESCRIPTION#~

Group:     Applications/Services
License:   RESERVED
URL:       https://%{gh_cvspath}/%{gh_project}

BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-%(%{__id_u} -n)

Provides:  %{gh_project} = %{version}

%description
~#SHORTDESCRIPTION#~

%build
(cd %{_current_directory} && make build)

%install
rm -rf $RPM_BUILD_ROOT
(cd %{_current_directory} && make install DESTDIR=$RPM_BUILD_ROOT)

%clean
rm -rf $RPM_BUILD_ROOT
(cd %{_current_directory} && make clean)

%files
%attr(-,root,root) %{_binpath}/%{_project}
%attr(-,root,root) %{_initpath}/%{_project}
%attr(-,root,root) %{_docpath}
%attr(-,root,root) %{_manpath}/%{_project}.1.gz
%docdir %{_docpath}
%docdir %{_manpath}
%config(noreplace) %{_configpath}*

%changelog
* Wed Nov 18 2015 ~#OWNER#~ <~#OWNEREMAIL#~> 1.0.0-1
- Initial Commit

