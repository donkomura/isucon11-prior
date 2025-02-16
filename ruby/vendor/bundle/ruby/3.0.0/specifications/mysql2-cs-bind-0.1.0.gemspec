# -*- encoding: utf-8 -*-
# stub: mysql2-cs-bind 0.1.0 ruby lib

Gem::Specification.new do |s|
  s.name = "mysql2-cs-bind".freeze
  s.version = "0.1.0"

  s.required_rubygems_version = Gem::Requirement.new(">= 0".freeze) if s.respond_to? :required_rubygems_version=
  s.require_paths = ["lib".freeze]
  s.authors = ["TAGOMORI Satoshi".freeze]
  s.date = "2020-09-15"
  s.description = "extension for mysql2 to add client-side variable binding, by adding method Mysql2::Client#xquery".freeze
  s.email = ["tagomoris@gmail.com".freeze]
  s.homepage = "https://github.com/tagomoris/mysql2-cs-bind".freeze
  s.required_ruby_version = Gem::Requirement.new(">= 2.0.0".freeze)
  s.rubygems_version = "3.2.15".freeze
  s.summary = "extension for mysql2 to add client-side variable binding".freeze

  s.installed_by_version = "3.2.15" if s.respond_to? :installed_by_version

  if s.respond_to? :specification_version then
    s.specification_version = 4
  end

  if s.respond_to? :add_runtime_dependency then
    s.add_runtime_dependency(%q<mysql2>.freeze, [">= 0"])
    s.add_development_dependency(%q<eventmachine>.freeze, [">= 0"])
    s.add_development_dependency(%q<rake-compiler>.freeze, ["~> 0.7.7"])
    s.add_development_dependency(%q<rake>.freeze, ["= 0.8.7"])
    s.add_development_dependency(%q<rspec>.freeze, ["= 2.10.0"])
  else
    s.add_dependency(%q<mysql2>.freeze, [">= 0"])
    s.add_dependency(%q<eventmachine>.freeze, [">= 0"])
    s.add_dependency(%q<rake-compiler>.freeze, ["~> 0.7.7"])
    s.add_dependency(%q<rake>.freeze, ["= 0.8.7"])
    s.add_dependency(%q<rspec>.freeze, ["= 2.10.0"])
  end
end
