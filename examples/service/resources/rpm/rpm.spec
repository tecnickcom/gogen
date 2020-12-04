# SPEC file

%global c_vendor    %{_vendor}
%global gh_owner    %{_owner}
%global gh_cvspath  %{_cvspath}
%global gh_project  %{_project}

Name:      %{_package}
Version:   %{_version}
Release:   %{_release}%{?dist}
Summary:   gosrvlibexampleshortdesc

Group:     Applications/Services
License:   %{_docpath}/LICENSE
URL:       https://%{gh_cvspath}/%{gh_project}

BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-%(%{__id_u} -n)

Provides:  %{gh_project} = %{version}

%description
gosrvlibexampleshortdesc

%build
#(cd %{_current_directory} && make build)

%install
rm -rf $RPM_BUILD_ROOT
(cd %{_current_directory} && make install DESTDIR=$RPM_BUILD_ROOT)

%clean
rm -rf $RPM_BUILD_ROOT

%files
%attr(-,root,root) %{_binpath}/%{_project}
%attr(-,root,root) %{_initpath}/%{_project}
%attr(-,root,root) %{_docpath}
%attr(-,root,root) %{_manpath}/%{_project}.1.gz
%docdir %{_docpath}
%docdir %{_manpath}
%config(noreplace) %{_configpath}*

%changelog
* Fri Dec 04 2020 gosrvlibexampleauthor <gosrvlibexampleemail> 1.0.0-1
- Initial Commit

